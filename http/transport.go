package httpsvc

import (
	"bytes"
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
	// Close the transport. If the transport support request draining, then the timeout specified will be the maximum
	// allowed time draining is allowed to take.
	Close(timeout time.Duration)
	// Sends a request to a remote server; note that the signature of this method makes it a Service.
	Send(Request) Response
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
			Transport: http.DefaultTransport,
			Timeout:   2 * time.Hour}}
}

func (t *networkTransport) Listen(name string, svc Service) error {
	l, err := t.ensureListening()
	if err != nil {
		return err
	}

	t.serverM.Lock()
	if t.server == nil {
		t.server = &http.Server{
			Addr:           t.addr,
			Handler:        t,
			MaxHeaderBytes: http.DefaultMaxHeaderBytes}
		go t.server.Serve(l)
	}
	t.serverM.Unlock()

	t.servicesM.Lock()
	t.services[name] = svc
	t.servicesM.Unlock()
	return nil
}

func (t *networkTransport) Close(timeout time.Duration) {
	t.listenerM.Lock()
	defer t.listenerM.Unlock()
	if t.listener != nil {
		// @TODO: Draining
		t.listener.Close()
		t.listener = nil
		log.Info(nil, "[Typhon:http:networkTransport] Stopped listening on %v", t.addr)
	}
}

func (t *networkTransport) Send(req Request) Response {
	log.Trace(req, "[Typhon:http:networkTransport] Sending to %s@%v", req.Host, req.URL)
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
		log.Info(nil, "[Typhon:http:networkTransport] Listening on %v", t.addr)
		t.listener = l
	}
	return t.listener, nil
}

func (t *networkTransport) ServeHTTP(rw http.ResponseWriter, httpReq *http.Request) {
	req := Request{
		Request: *httpReq,
		Context: context.Background()} // @TODO: Proper context
	log.Trace(req, "[Typhon:http:networkTransport] Received for %s", req.Host)

	t.servicesM.RLock()
	svc, ok := t.services[req.Host]
	t.servicesM.RUnlock()
	if ok {
		rsp := svc(req)
		rsp.Write(rw)
	} else {
		panic("not handled")
	}
}
