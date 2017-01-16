package typhon

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/facebookgo/httpdown"
	log "github.com/monzo/slog"
)

const DefaultListenAddr = "127.0.0.1:0"

type Server interface {
	httpdown.Server
	Listener() net.Listener
	WaitC() <-chan struct{}
}

type server struct {
	httpdown.Server
	l net.Listener
}

func (s server) Listener() net.Listener {
	return s.l
}

func (s server) WaitC() <-chan struct{} {
	c := make(chan struct{}, 0)
	go func() {
		s.Server.Wait()
		close(c)
	}()
	return c
}

func Serve(svc Service, l net.Listener) (Server, error) {
	downer := &httpdown.HTTP{
		StopTimeout: 20 * time.Second,
		KillTimeout: 25 * time.Second}
	downerServer := downer.Serve(HttpServer(svc), l)
	log.Info(nil, "Listening on %v", l.Addr())
	return server{
		Server: downerServer,
		l:      l}, nil
}

func Listen(svc Service, addr string) (Server, error) {
	// Determine on which address to listen, choosing in order one of:
	// 1. The passed addr
	// 2. PORT variable (listening on all interfaces)
	// 3. Random, available port, on the loopback interface only
	if addr == "" {
		if _addr := os.Getenv("LISTEN_ADDR"); _addr != "" {
			addr = _addr
		} else if port, err := strconv.Atoi(os.Getenv("PORT")); err == nil && port >= 0 {
			addr = fmt.Sprintf(":%d", port)
		} else {
			addr = ":0"
		}
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}

	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}
	return Serve(svc, l)
}
