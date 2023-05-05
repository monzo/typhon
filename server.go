package typhon

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/monzo/slog"
)

type Server struct {
	l              net.Listener
	srv            *http.Server
	shuttingDown   chan struct{}
	shutdownOnce   sync.Once
	shutdownFuncs  []func(context.Context)
	shutdownFuncsM sync.Mutex
}

// ServerOption allows customizing the underling http.Server
type ServerOption func(*Server)

// Listener returns the network listener that this server is active on.
func (s *Server) Listener() net.Listener {
	return s.l
}

// Done returns a channel that will be closed when the server begins to shutdown. The server may still be draining its
// connections at the time the channel is closed.
func (s *Server) Done() <-chan struct{} {
	return s.shuttingDown
}

// Stop shuts down the server, returning when there are no more connections still open. Graceful shutdown will be
// attempted until the passed context expires, at which time all connections will be forcibly terminated.
func (s *Server) Stop(ctx context.Context) {
	s.shutdownFuncsM.Lock()
	defer s.shutdownFuncsM.Unlock()
	s.shutdownOnce.Do(func() {
		close(s.shuttingDown)
		// Shut down the HTTP server in parallel to calling any custom shutdown functions
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.srv.Shutdown(ctx); err != nil {
				slog.Debug(ctx, "Graceful shutdown failed; forcibly closing connections 👢")
				if err := s.srv.Close(); err != nil {
					slog.Critical(ctx, "Forceful shutdown failed, exiting 😱: %v", err)
					panic(err) // Something is super hosed here
				}
			}
		}()
		for _, f := range s.shutdownFuncs {
			f := f // capture range variable
			wg.Add(1)
			go func() {
				defer wg.Done()
				f(ctx)
			}()
		}
		wg.Wait()
	})
}

// addShutdownFunc registers a function that will be called when the server is stopped. The function is expected to try
// to shutdown gracefully until the context expires, at which time it should terminate its work forcefully.
func (s *Server) addShutdownFunc(f func(context.Context)) {
	s.shutdownFuncsM.Lock()
	defer s.shutdownFuncsM.Unlock()
	s.shutdownFuncs = append(s.shutdownFuncs, f)
}

// Serve starts a HTTP server, binding the passed Service to the passed listener and applying the passed ServerOptions.
func Serve(svc Service, l net.Listener, opts ...ServerOption) (*Server, error) {
	s := &Server{
		l:            l,
		shuttingDown: make(chan struct{})}
	svc = svc.Filter(func(req Request, svc Service) Response {
		req.server = s
		return svc(req)
	})
	s.srv = &http.Server{
		Handler:        HttpHandler(svc),
		MaxHeaderBytes: http.DefaultMaxHeaderBytes,
	}

	// Apply any given ServerOptions
	for _, opt := range opts {
		opt(s)
	}

	go func() {
		err := s.srv.Serve(l)
		if err != nil && err != http.ErrServerClosed {
			slog.Error(nil, "HTTP server error: %v", err)
			// Stopping with an already-closed context means we go immediately to "forceful" mode
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			s.Stop(ctx)
		}
	}()
	return s, nil
}

func Listen(svc Service, addr string, opts ...ServerOption) (*Server, error) {
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
	return Serve(svc, l, opts...)
}

// TimeoutOptions specifies various server timeouts. See http.Server for details of what these do.
// There's a nice post explaining them here: https://ieftimov.com/posts/make-resilient-golang-net-http-servers-using-timeouts-deadlines-context-cancellation/#server-timeouts---first-principles
// WARNING: Due to a Go bug, connections using h2c do not respect these timeouts.
// See https://github.com/golang/go/issues/52868
type TimeoutOptions struct {
	Read       time.Duration
	ReadHeader time.Duration
	Write      time.Duration
	Idle       time.Duration
}

// WithTimeout sets the server timeouts.
func WithTimeout(opts TimeoutOptions) ServerOption {
	return func(s *Server) {
		s.srv.ReadTimeout = opts.Read
		s.srv.ReadHeaderTimeout = opts.ReadHeader
		s.srv.WriteTimeout = opts.Write
		s.srv.IdleTimeout = opts.Idle
	}
}

var (
	connectionStartTimeHeaderKey = "X-Typhon-Connection-Start"
	// addConnectionStartTimeHeader is set to true within tests to
	// make it easier to test the server option.
	addConnectionStartTimeHeader = false
)

// WithMaxConnectionAge returns a server option that will enforce a max
// connection age. When a connection has reached the max connection age
// then the next request that is processed on that connection will result
// in the connection being gracefully closed. This does mean that if a
// connection is not being used then it can outlive the maximum connection
// age.
func WithMaxConnectionAge(maxAge time.Duration) ServerOption {
	// We have no ability within a handler to get access to the
	// underlying net.Conn that the request came on. However,
	// the http.Server has a ConnContext field that can be used
	// to specify a function that can modify the context used for
	// that connection. We can use this to store the connection
	// start time in the context and then in the handler we can
	// read that out and whenever the maxAge has been exceeded we
	// can close the connection.
	//
	// We could close the connection by calling the Close method
	// on the net.Conn. This would have the benefit that we could
	// close the connection exactly at the expiry but would have
	// the disadvantage that it does not gracefully close the
	// connection – it would kill all in-flight requests. Instead,
	// we set the 'Connection: close' response header which will
	// be translated into an HTTP2 GOAWAY frame and result in the
	// connection being gracefully closed.

	return func(s *Server) {
		// Wrap the current ConnContext (if set) to store a reference
		// to the connection start time in the context.
		origConnContext := s.srv.ConnContext
		s.srv.ConnContext = func(ctx context.Context, conn net.Conn) context.Context {
			if origConnContext != nil {
				ctx = origConnContext(ctx, conn)
			}

			return setConnectionStartTimeInContext(ctx, time.Now())
		}

		// Wrap the handler to set the 'Connection: close' response
		// header if the max age has been exceeded.
		origHandler := s.srv.Handler
		s.srv.Handler = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			connectionStart, ok := readConnectionStartTimeFromContext(request.Context())
			if ok {
				if time.Since(connectionStart) > maxAge {
					h := writer.Header()
					h.Add("Connection", "close")
				}

				// This is used within tests
				if addConnectionStartTimeHeader {
					h := writer.Header()
					h.Add(connectionStartTimeHeaderKey, connectionStart.String())
				}
			}

			origHandler.ServeHTTP(writer, request)
		})
	}
}

type connectionContextKey struct{}

func setConnectionStartTimeInContext(parent context.Context, t time.Time) context.Context {
	return context.WithValue(parent, connectionContextKey{}, t)
}

func readConnectionStartTimeFromContext(ctx context.Context) (time.Time, bool) {
	conn, ok := ctx.Value(connectionContextKey{}).(time.Time)
	return conn, ok
}
