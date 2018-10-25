package typhon

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/textproto"
	"sync"

	"github.com/monzo/terrors"
	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// H2cFilter adds HTTP/2 h2c upgrade support to the wrapped Service (as defined in RFC 7540 §3.2, §3.4).
func H2cFilter(req Request, svc Service) Response {
	h := req.Header
	// h2c with prior knowledge (RFC 7540 Section 3.4)
	isPrior := (req.Method == "PRI" && len(h) == 0 && req.URL.Path == "*" && req.Proto == "HTTP/2.0")
	// h2c upgrade (RFC 7540 Section 3.2)
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
// connection draining, and all the hooks that make would make this easy are unexported.
//
// If this ticket gets resolved this code can be dramatically simplified, but it is not a priority for the Go team:
// https://github.com/golang/go/issues/26682
//
// 🤢

var h2cConns sync.Map // map[*Server]*h2cInfo

// h2cInfo stores information about connections that have been upgraded by a single Typhon server
type h2cInfo struct {
	sync.Mutex
	conns []*hijackedConn
	h2s   *http2.Server
}

// hijackedConn represents a network connection that has been hijacked for a h2c upgrade. This is necessary because we
// need to know when the connection has been closed, to know if/when graceful shutdown completes.
type hijackedConn struct {
	net.Conn
	closed    chan struct{}
	closeOnce sync.Once
}

func (c *hijackedConn) Close() error {
	defer c.closeOnce.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

type h2cHijacker struct {
	http.ResponseWriter
	http.Hijacker
	hijacked func(*hijackedConn)
}

func (h h2cHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	c, r, err := h.Hijacker.Hijack()
	conn := &hijackedConn{
		Conn:   c,
		closed: make(chan struct{})}
	h.hijacked(conn)
	return conn, r, err
}

func shutdownH2c(ctx context.Context, srv *Server) {
	_h2c, ok := h2cConns.Load(srv)
	if !ok {
		return
	}
	h2c := _h2c.(*h2cInfo)
	h2c.Lock()
	defer h2c.Unlock()

gracefulCloseLoop:
	for _, c := range h2c.conns {
		select {
		case <-ctx.Done():
			break gracefulCloseLoop
		case <-c.closed:
			h2c.conns = h2c.conns[1:]
		}
	}
	// If any connections remain after gracefulCloseLoop, we need to forcefully close them
	for _, c := range h2c.conns {
		c.Close()
		h2c.conns = h2c.conns[1:]
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
		h2s: &http2.Server{}}
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
			shutdownH2c(ctx, srv)
		})
	}

	h := h2cHijacker{
		ResponseWriter: rw,
		Hijacker:       hijacker,
		hijacked: func(c *hijackedConn) {
			h2c.Lock()
			defer h2c.Unlock()
			h2c.conns = append(h2c.conns, c)
		}}
	return h, h2c.h2s, nil
}
