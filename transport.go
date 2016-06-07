package typhon

// import (
// 	"bytes"
// 	"encoding/json"
// 	"errors"
// 	"fmt"
// 	"io"
// 	"io/ioutil"
// 	"net"
// 	"net/http"
// 	"os"
// 	"strconv"
// 	"sync"
// 	"time"
//
// 	"github.com/facebookgo/httpcontrol"
// 	log "github.com/mondough/slog"
// 	"github.com/mondough/terrors"
// 	terrorp "github.com/mondough/terrors/proto"
// 	"golang.org/x/net/context"
// )
//
// type errReader struct {
// 	error
// }
//
// func (e errReader) Read(_ []byte) (int, error) {
// 	return 0, e
// }
//
// type Transport struct {
// 	listenerM sync.RWMutex       // protects listener
// 	listener  *net.TCPListener   // initialised on first Listen()
// 	servicesM sync.RWMutex       // protects services
// 	services  map[string]Service // maps service names to services
// 	client    *http.Client       // immutable (no mutex needed)
// 	serverM   sync.Mutex         // protects server
// 	server    *http.Server       // initialised on first Listen()
// 	Send      Service
// }
//
// // A single HTTP client is used among all transports
// var sharedHttpClient = &http.Client{
// 	Timeout: time.Hour,
// 	Transport: &httpcontrol.Transport{
// 		Proxy:                 http.ProxyFromEnvironment,
// 		DisableKeepAlives:     false,
// 		DisableCompression:    false,
// 		MaxIdleConnsPerHost:   10,
// 		DialTimeout:           10 * time.Second,
// 		DialKeepAlive:         10 * time.Minute,
// 		ResponseHeaderTimeout: time.Minute,
// 		RequestTimeout:        time.Hour,
// 		RetryAfterTimeout:     false,
// 		MaxTries:              3}}
//
// func NetworkTransport() *Transport {
// 	sharedTransportOnce.Do(func() {
// 		sharedTransport = &Transport{
// 			services: make(map[string]Service, 1),
// 			client:   sharedHttpClient}
// 		sharedTransport.Send = sharedTransport.send
// 	})
// 	return sharedTransport
// }
//
// func (t *Transport) Listen(name string, svc Service) error {
// 	l, err := t.ensureListening()
// 	if err != nil {
// 		return err
// 	}
//
// 	t.servicesM.Lock()
// 	t.services[name] = svc
// 	t.servicesM.Unlock()
//
// 	t.serverM.Lock()
// 	if t.server == nil {
// 		t.server = &http.Server{
// 			Handler:        t,
// 			MaxHeaderBytes: http.DefaultMaxHeaderBytes}
// 		go func() {
// 			err := t.server.Serve(l)
// 			log.Info(nil, "[Typhon:http:networkTransport] Server exited with %v", err)
// 		}()
// 	}
// 	t.serverM.Unlock()
// 	return nil
// }
//
// func (t *Transport) RemoteAddr() net.Addr {
// 	t.listenerM.RLock()
// 	defer t.listenerM.RUnlock()
// 	if t.listener != nil {
// 		return t.listener.Addr()
// 	}
// 	return nil
// }
//
// func (t *Transport) Unlisten(name string) {
// 	// @TODO: Draining
// 	t.servicesM.Lock()
// 	delete(t.services, name)
// 	t.servicesM.Unlock()
// }
//
// func (t *Transport) Close(timeout time.Duration) {
// 	t.listenerM.Lock()
// 	defer t.listenerM.Unlock()
// 	if t.listener != nil {
// 		t.servicesM.RLock()
// 		svcNames := make([]string, 0, len(t.services))
// 		for k, _ := range t.services {
// 			svcNames = append(svcNames, k)
// 		}
// 		t.servicesM.RUnlock()
// 		for _, svc := range svcNames {
// 			t.Unlisten(svc)
// 		}
// 		addr := t.listener.Addr()
// 		t.listener.Close()
// 		t.listener = nil
// 		log.Info(nil, "[Typhon:http:networkTransport] Stopped listening on %v", addr)
// 	}
// }
//
// func (t *Transport) send(req Request) Response {
// 	httpRsp, err := t.client.Do(&(req.Request))
// 	ret := Response{
// 		Error: err}
// 	if httpRsp != nil {
// 		defer httpRsp.Body.Close()
// 		ret.Response = httpRsp
// 		// Read the entire response body here so we can make sure the request is Close()d
// 		// @TODO: Provide a hook to allow the client to get a streaming body?
// 		payload := new(bytes.Buffer)
// 		_, err := io.Copy(payload, httpRsp.Body)
// 		if err != nil {
// 			ret.Response.Body = ioutil.NopCloser(errReader{err})
// 		} else {
// 			ret.Response.Body = ioutil.NopCloser(payload)
//
// 			// Errors should set an error on the response; in the
// 			if ret.StatusCode >= 400 && ret.StatusCode < 600 {
// 				if ret.Header.Get("Terror") == "1" {
// 					terrp := &terrorp.Error{}
// 					if err := json.Unmarshal(payload.Bytes(), terrp); err == nil {
// 						ret.Error = terrors.Unmarshal(terrp)
// 					} else {
//
// 						ret.Error = errors.New(string(payload.Bytes()))
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return ret
// }
//
// func (t *Transport) ensureListening() (net.Listener, error) {
// 	// Determine on which address to listen, choosing in order one of:
// 	// 1. LISTEN_ADDR environment variable
// 	// 2. PORT variable (listening on all interfaces)
// 	// 3. Random, available port
// 	addr := ":0"
// 	if addr_ := os.Getenv("LISTEN_ADDR"); addr_ != "" {
// 		addr = addr_
// 	} else if port, err := strconv.Atoi(os.Getenv("PORT")); err == nil && port >= 0 {
// 		addr = fmt.Sprintf(":%d", port)
// 	}
//
// 	t.listenerM.Lock()
// 	defer t.listenerM.Unlock()
// 	if t.listener == nil {
// 		tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
// 		if err != nil {
// 			return nil, terrors.Wrap(err, nil)
// 		}
//
// 		// @TODO: Keep alives
// 		l, err := net.ListenTCP("tcp", tcpAddr)
// 		if err != nil {
// 			return nil, terrors.Wrap(err, nil)
// 		}
// 		log.Info(nil, "[Typhon:http:networkTransport] Listening on %v", l.Addr())
// 		t.listener = l
// 	}
// 	return t.listener, nil
// }
//
// func (t *Transport) ServeHTTP(rw http.ResponseWriter, httpReq *http.Request) {
// 	req := Request{
// 		Request: *httpReq,
// 		Context: context.Background()} // @TODO: Proper context
// 	rsp := Response{}
//
// 	t.servicesM.RLock()
// 	svc, ok := t.services[req.Host]
// 	t.servicesM.RUnlock()
// 	if ok {
// 		rsp = svc(req)
// 	} else {
// 		rsp.Error = terrors.InternalService("unhandled_service", fmt.Sprintf("Unhandled service %s", req.Host), nil)
// 	}
//
// 	rsp.ctx = req
// 	if rsp.Response == nil { // Error responses often won't have a HTTP response
// 		rsp.Response = newHttpResponse(req)
// 	}
//
// 	// If the response is an error, serialise it
// 	if rsp.Error != nil {
// 		terr := terrors.Wrap(rsp.Error, nil).(*terrors.Error)
// 		rsp.StatusCode = terr2StatusCode(terr.Code)
// 		rsp.Encode(terrors.Marshal(terr))
// 		rsp.Header.Set("Terror", "1")
// 	}
//
// 	// Write the response out to the wire
// 	for k, v := range rsp.Header {
// 		rw.Header()[k] = v
// 	}
// 	rw.WriteHeader(rsp.StatusCode)
// 	if rsp.Body != nil {
// 		defer rsp.Body.Close()
// 		if _, err := io.Copy(rw, rsp.Body); err != nil {
// 			log.Error(req, "[Typhon:http:networkTransport] Error copying response body: %v", err)
// 		}
// 	}
// }
