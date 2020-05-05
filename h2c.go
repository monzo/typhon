package typhon

import (
	"bufio"
	"context"
	"math"
	"net"
	"net/http"
	"net/textproto"
	"sync"

	"github.com/deckarep/golang-set"
	"github.com/monzo/terrors"
	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// H2cFilter adds HTTP/2 h2c upgrade support to the wrapped Service (as defined in RFC 7540 §3.2, §3.4).
func H2cFilter(req Request, svc Service) Response {
	h := req.Header
	// h2c with prior knowledge (RFC 7540 §3.4)
	isPrior := (req.Method == "PRI" && len(h) == 0 && req.URL.Path == "*" && req.Proto == "HTTP/2.0")
	// h2c upgrade (RFC 7540 §3.2)
	isUpgrade := httpguts.HeaderValuesContainsToken(h[textproto.CanonicalMIMEHeaderKey("Upgrade")], "h2c") &&
		httpguts.HeaderValuesContainsToken(h[textproto.CanonicalMIMEHeaderKey("Connection")], "HTTP2-Settings")
	if isPrior || isUpgrade {
		rsp := NewResponse(req)
		rw, h2s, err := setupH2cHijacker(req, rsp.Writer())
		if err != nil {
			return Response{Error: err}
		}
		h2c.NewHandler(HttpHandler(svc), h2s).ServeHTTP(rw, &req.Request)
		return rsp
	}
	return svc(req)
}

// Dear reader: I'm sorry, the code below isn't fun. This is because Go's h2c implementation doesn't have support for
// connection draining, and all the hooks that would make this easy are unexported.
//
// If this ticket gets resolved this code can be dramatically simplified, but it is not a priority for the Go team:
// https://github.com/golang/go/issues/26682
//
// 🤢

var h2cConns sync.Map // map[*Server]*h2cInfo

// h2cInfo stores information about connections that have been upgraded by a single Typhon server
type h2cInfo struct {
	conns mapset.Set
	h2s   *http2.Server
}

// hijackedConn represents a network connection that has been hijacked for a h2c upgrade. This is necessary because we
// need to know when the connection has been closed, to know if/when graceful shutdown completes.
type hijackedConn struct {
	net.Conn
	onClose   func(*hijackedConn)
	closed    chan struct{}
	closeOnce sync.Once
}

func (c *hijackedConn) Close() error {
	defer c.closeOnce.Do(func() {
		close(c.closed)
		c.onClose(c)
	})
	return c.Conn.Close()
}

type h2cHijacker struct {
	http.ResponseWriter
	http.Hijacker
	onHijack func(*hijackedConn)
}

func (h h2cHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c, r, err := h.Hijacker.Hijack()
	conn := &hijackedConn{
		Conn:   c,
		closed: make(chan struct{})}
	h.onHijack(conn)
	return conn, r, err
}

func shutdownH2c(ctx context.Context, srv *Server, h2c *h2cInfo) {
gracefulCloseLoop:
	for _, _c := range h2c.conns.ToSlice() {
		c := _c.(*hijackedConn)
		select {
		case <-ctx.Done():
			break gracefulCloseLoop
		case <-c.closed:
			h2c.conns.Remove(c)
		}
	}
	// If any connections remain after gracefulCloseLoop, we need to forcefully close them
	for _, _c := range h2c.conns.ToSlice() {
		c := _c.(*hijackedConn)
		c.Close()
		h2c.conns.Remove(c)
	}
	h2cConns.Delete(srv)
}

func setupH2cHijacker(req Request, rw http.ResponseWriter) (http.ResponseWriter, *http2.Server, error) {
	hijacker, ok := rw.(http.Hijacker)
	if !ok {
		err := terrors.InternalService("hijack_impossible", "Cannot hijack response; h2c upgrade impossible", nil)
		return nil, nil, err
	}
	srv := req.server
	if srv == nil {
		return rw, &http2.Server{}, nil
	}

	h2c := &h2cInfo{
		conns: mapset.NewSet(),
		h2s: &http2.Server{
			// We're copying envoy and grpc by setting this to the max uint32.
			// The Go default is 250 which is not ideal for long lived streaming requests.
			MaxConcurrentStreams: math.MaxUint32,
		}}
	_h2c, loaded := h2cConns.LoadOrStore(srv, h2c)
	h2c = _h2c.(*h2cInfo)
	if !loaded {
		// http2.ConfigureServer wires up an unexported method within the http2 library so it gracefully drains h2c
		// connections when the http1 server is stopped. However, this happens asynchronously: the http1 server will
		// think it has shut down before the h2c connections have finished draining. To work around this, we add
		// a shutdown function of our own in the Typhon server which waits for connections to be drained, or if things
		// timeout before then to terminate them forcefully.
		http2.ConfigureServer(srv.srv, h2c.h2s)
		srv.addShutdownFunc(func(ctx context.Context) {
			shutdownH2c(ctx, srv, h2c)
		})
	}

	h := h2cHijacker{
		ResponseWriter: rw,
		Hijacker:       hijacker,
		onHijack: func(c *hijackedConn) {
			h2c.conns.Add(c)
			// when the connection closes, remove from h2cInfo's to prevent refs to dead connections accumulating
			c.onClose = func(c *hijackedConn) {
				h2c.conns.Remove(c)
			}
		}}
	return h, h2c.h2s, nil
}
