package httpsvc

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/facebookgo/httpcontrol"
	log "github.com/mondough/slog"
	"github.com/mondough/terrors"
	"golang.org/x/net/context"
)

type errReader struct {
	error
}

func (e errReader) Read(_ []byte) (int, error) {
	return 0, e
}

type Transport interface {
	// Accept inbound requests for the service, identified by name.
	Listen(name string, svc Service) error
	Unlisten(name string)
	// Close the transport. If the transport support request draining, then the timeout specified will be the maximum
	// allowed time draining is allowed to take.
	Close(timeout time.Duration)
	// Sends a request to a remote server; note that the signature of this method makes it a Service.
	Send(Request) Response
	// RemoteAddr returns the address at which this transport can be reached by clients.
	RemoteAddr() net.Addr
}

type networkTransport struct {
	listenerM    sync.RWMutex       // protects listener and listenerDone
	listener     *net.TCPListener   // initialised on first Listen()
	listenerDone <-chan struct{}    // closed when the listener terminates
	servicesM    sync.RWMutex       // protects services
	services     map[string]Service // maps service names to services
	client       *http.Client       // immutable (no mutex needed)
	serverM      sync.Mutex         // protects server
	server       *http.Server       // initialised on first Listen()
}

// A single transport is shared
var sharedTransport = &networkTransport{
	services: make(map[string]Service, 1),
	client: &http.Client{
		Timeout: time.Hour,
		Transport: &httpcontrol.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DisableKeepAlives:     false,
			DisableCompression:    false,
			MaxIdleConnsPerHost:   10,
			DialTimeout:           10 * time.Second,
			DialKeepAlive:         10 * time.Minute,
			ResponseHeaderTimeout: time.Minute,
			RequestTimeout:        time.Hour,
			RetryAfterTimeout:     false,
			MaxTries:              3}}}

func NetworkTransport() Transport {
	return sharedTransport
}

func (t *networkTransport) Listen(name string, svc Service) error {
	l, err := t.ensureListening()
	if err != nil {
		return err
	}

	t.servicesM.Lock()
	t.services[name] = svc
	t.servicesM.Unlock()

	t.serverM.Lock()
	if t.server == nil {
		t.server = &http.Server{
			Handler:        t,
			MaxHeaderBytes: http.DefaultMaxHeaderBytes}
		go func() {
			err := t.server.Serve(l)
			log.Info(nil, "[Typhon:http:networkTransport] Server exited with %v", err)
		}()
	}
	t.serverM.Unlock()
	return nil
}

func (t *networkTransport) RemoteAddr() net.Addr {
	t.listenerM.RLock()
	defer t.listenerM.RUnlock()
	if t.listener != nil {
		return t.listener.Addr()
	}
	return nil
}

func (t *networkTransport) Unlisten(name string) {
	// @TODO: Draining
	t.servicesM.Lock()
	delete(t.services, name)
	t.servicesM.Unlock()
}

func (t *networkTransport) Close(timeout time.Duration) {
	t.listenerM.Lock()
	defer t.listenerM.Unlock()
	if t.listener != nil {
		t.servicesM.RLock()
		svcNames := make([]string, 0, len(t.services))
		for k, _ := range t.services {
			svcNames = append(svcNames, k)
		}
		t.servicesM.RUnlock()
		for _, svc := range svcNames {
			t.Unlisten(svc)
		}
		addr := t.listener.Addr()
		t.listener.Close()
		t.listener = nil
		log.Info(nil, "[Typhon:http:networkTransport] Stopped listening on %v", addr)
	}
}

func (t *networkTransport) Send(req Request) Response {
	httpRsp, err := t.client.Do(&(req.Request))
	ret := Response{
		Error: err}
	if httpRsp != nil {
		defer httpRsp.Body.Close()
		ret.Response = httpRsp
		// Read the entire response body here so we can make sure the request is Close()d
		// @TODO: Provide a hook to allow the client to get a streaming body?
		payload := new(bytes.Buffer)
		_, err := io.Copy(payload, httpRsp.Body)
		if err != nil {
			ret.Response.Body = ioutil.NopCloser(errReader{err})
		} else {
			ret.Response.Body = ioutil.NopCloser(payload)
		}
	}
	return ret
}

func (t *networkTransport) ensureListening() (net.Listener, error) {
	// Determine on which address to listen, choosing in order one of:
	// 1. LISTEN_ADDR environment variable
	// 2. PORT variable (listening on all interfaces)
	// 3. Random, available port
	addr := ":0"
	if addr_ := os.Getenv("LISTEN_ADDR"); addr_ != "" {
		addr = addr_
	} else if port, err := strconv.Atoi(os.Getenv("PORT")); err == nil && port >= 0 {
		addr = fmt.Sprintf(":%d", port)
	}

	t.listenerM.Lock()
	defer t.listenerM.Unlock()
	if t.listener == nil {
		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			return nil, terrors.Wrap(err, nil)
		}

		// @TODO: Keep alives
		l, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return nil, terrors.Wrap(err, nil)
		}
		log.Info(nil, "[Typhon:http:networkTransport] Listening on %v", l.Addr())
		t.listener = l
	}
	return t.listener, nil
}

func (t *networkTransport) ServeHTTP(rw http.ResponseWriter, httpReq *http.Request) {
	req := Request{
		Request: *httpReq,
		Context: context.Background()} // @TODO: Proper context
	rsp := Response{}

	t.servicesM.RLock()
	svc, ok := t.services[req.Host]
	t.servicesM.RUnlock()
	if ok {
		rsp = svc(req)
	} else {
		rsp.Error = terrors.InternalService("unhandled_service", fmt.Sprintf("Unhandled service %s", req.Host), nil)
	}
	rsp.WriteTo(req, rw)
}
