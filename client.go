package typhon

import (
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/mondough/terrors"
	"golang.org/x/net/context"
)

var (
	// Client is used to send all requests by default. It can be overridden globally but MUST only be done before use
	// takes place; access is not synchronised.
	Client              Service = BareClient
	httpClientTransport         = &httpcontrol.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DisableKeepAlives:     false,
		DisableCompression:    false,
		MaxIdleConnsPerHost:   10,
		DialTimeout:           10 * time.Second,
		DialKeepAlive:         10 * time.Minute,
		ResponseHeaderTimeout: time.Minute,
		RequestTimeout:        time.Hour,
		RetryAfterTimeout:     false,
		MaxTries:              6}
	httpClient = &http.Client{ // shared for all outbound requests
		Timeout:   time.Hour,
		Transport: httpClientTransport}
)

type ResponseFuture struct {
	r      Response
	cancel context.CancelFunc
	mtx    sync.RWMutex
	done   <-chan struct{}
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
	f.cancel()
}

func BareClient(req Request) Response {
	// If the request is cancelled, cancel the request
	// Go 1.7's http library has native context support, so this can go away
	httpReq := &req.Request
	done := make(chan struct{})
	cancel := make(chan struct{})
	superCancel := (<-chan struct{})(cancel)
	if httpReq.Cancel != nil { // If there already was a cancel channel, wrap it
		superCancel = httpReq.Cancel
	}
	httpReq.Cancel = cancel
	go func() {
		select {
		case <-done:
		case <-httpReq.Cancel:
		case <-superCancel:
			close(cancel)
		case <-req.Context.Done():
			close(cancel)
		}
	}()

	httpRsp, err := httpClient.Do(httpReq)
	close(done)

	// Read the response in its entirety and close the Response body here; this protects us from callers that forget to
	// call Close() but does not allow streaming responses.
	// @TODO: Streaming client?
	if httpRsp != nil && httpRsp.Body != nil {
		buf := &bufCloser{}
		io.Copy(buf, httpRsp.Body)
		httpRsp.Body.Close()
		httpRsp.Body = ioutil.NopCloser(buf)
	}

	return Response{
		Response: httpRsp,
		Error:    terrors.Wrap(err, nil)}
}

func SendVia(req Request, svc Service) *ResponseFuture {
	ctx, cancel := context.WithCancel(req.Context)
	req.Context = ctx
	done := make(chan struct{})
	f := &ResponseFuture{
		cancel: cancel,
		done:   done}
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
