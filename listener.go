package typhon

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/facebookgo/httpdown"
	log "github.com/mondough/slog"
	"golang.org/x/net/context"
)

var DefaultListenAddr = ":0"

type Listener interface {
	httpdown.Server
	Addr() net.Addr
	WaitC() <-chan struct{}
}

type listener struct {
	httpdown.Server
	addr net.Addr
}

func (l listener) Addr() net.Addr {
	return l.addr
}

func (l listener) WaitC() <-chan struct{} {
	c := make(chan struct{}, 0)
	go func() {
		l.Wait()
		close(c)
	}()
	return c
}

func Listen(svc Service) Listener {
	// Determine on which address to listen, choosing in order one of:
	// 1. LISTEN_ADDR environment variable
	// 2. PORT variable (listening on all interfaces)
	// 3. Random, available port
	addr := DefaultListenAddr
	if addr_ := os.Getenv("LISTEN_ADDR"); addr_ != "" {
		addr = addr_
	} else if port, err := strconv.Atoi(os.Getenv("PORT")); err == nil && port >= 0 {
		addr = fmt.Sprintf(":%d", port)
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		panic(err)
	}

	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		panic(err)
	}

	svc = svc.Filter(networkFilter)
	downer := &httpdown.HTTP{
		StopTimeout: 20 * time.Second,
		KillTimeout: 25 * time.Second}
	server := downer.Serve(&http.Server{
		Handler:        httpHandler(svc),
		MaxHeaderBytes: http.DefaultMaxHeaderBytes}, l)
	log.Info(nil, "Listening on %v", l.Addr())
	return listener{
		Server: server,
		addr:   l.Addr()}
}

func httpHandler(svc Service) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, httpReq *http.Request) {
		req := Request{
			Request: *httpReq,
			Context: context.Background()} // @TODO: Proper context
		rsp := svc(req)

		// Write the response out to the wire
		for k, v := range rsp.Header {
			rw.Header()[k] = v
		}
		rw.WriteHeader(rsp.StatusCode)
		if rsp.Body != nil {
			defer rsp.Body.Close()
			if _, err := io.Copy(rw, rsp.Body); err != nil {
				log.Error(req, "[Typhon:http:networkTransport] Error copying response body: %v", err)
			}
		}
	})
}
