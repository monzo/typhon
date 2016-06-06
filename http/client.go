package httpsvc

import (
	"net/http"
	"sync"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/mondough/terrors"
)

// Client is used to send all requests by default. It can be overridden globally but MUST only be done before use takes
// place; access is not synchronised.
var Client Service = BareClient

// shared for all outbound requests
var httpClient = &http.Client{
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
		MaxTries:              6}}

type ResponseFuture struct {
	r    Response
	mtx  sync.RWMutex
	done <-chan struct{}
}

func (f *ResponseFuture) WaitC() <-chan struct{} {
	return f.done
}

func (f *ResponseFuture) Response() Response {
	<-f.WaitC()
	f.mtx.RLock()
	defer f.mtx.RUnlock()
	return f.r
}

func (f *ResponseFuture) Cancel() {
	// @TODO: Implement me
}

func BareClient(req Request) Response {
	return networkFilter(req, func(req Request) Response {
		httpRsp, err := httpClient.Do(&req.Request)
		return Response{
			Response: httpRsp,
			Error:    terrors.Wrap(err, nil)}
	})
}

func SendVia(req Request, svc Service) *ResponseFuture {
	done := make(chan struct{})
	f := &ResponseFuture{
		done: done}
	go func() {
		defer close(done)
		rsp := svc(req)
		f.mtx.RLock()
		defer f.mtx.RUnlock()
		f.r = rsp
	}()
	return f
}

func Send(req Request) *ResponseFuture {
	return SendVia(req, Client)
}
