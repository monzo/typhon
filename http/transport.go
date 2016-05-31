package httpsvc

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

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
	addr      string             // immutable; set on creation
	listenerM sync.Mutex         // protects listener
	listener  *net.TCPListener   // initialised on first Listen()
	servicesM sync.RWMutex       // protects services
	services  map[string]Service // maps service names to services
	client    *http.Client       // immutable (no mutex needed)
	serverM   sync.Mutex         // protects server
	server    *http.Server       // initialised on first Listen()
}

func NetworkTransport(addr string) Transport {
	return &networkTransport{
		addr:     addr,
		services: make(map[string]Service, 1),
		client: &http.Client{
			Timeout: 2 * time.Hour,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				TLSHandshakeTimeout:   2 * time.Second,
				ExpectContinueTimeout: time.Second,
				MaxIdleConnsPerHost:   10,
				Dial: (&dialer{
					MaxConnsPerHost: 10,
					Dialer: net.Dialer{
						Timeout:   2 * time.Second,
						KeepAlive: 5 * time.Minute,
					}}).Dial}}}
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
			Addr:           t.addr,
			Handler:        t,
			MaxHeaderBytes: http.DefaultMaxHeaderBytes}
		go t.server.Serve(l)
	}
	t.serverM.Unlock()
	return nil
}

func (t *networkTransport) RemoteAddr() net.Addr {
	t.listenerM.Lock()
	defer t.listenerM.Unlock()
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
		ret.Response = *httpRsp
		// Read the entire response body here so we can make sure the request is Close()d
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
	t.listenerM.Lock()
	defer t.listenerM.Unlock()
	if t.listener == nil {
		addr, err := net.ResolveTCPAddr("tcp", t.addr)
		if err != nil {
			return nil, terrors.Wrap(err, nil)
		}

		// @TODO: Keep alives
		l, err := net.ListenTCP("tcp", addr)
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

	t.servicesM.RLock()
	svc, ok := t.services[req.Host]
	t.servicesM.RUnlock()
	if ok {
		rsp := svc(req)
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
	} else {
		rw.WriteHeader(http.StatusGatewayTimeout)
		fmt.Fprintf(rw, "Unhandled service %s", req.Host)
	}
}
