package typhon

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/facebookgo/httpcontrol"
	log "github.com/mondough/slog"
	"github.com/mondough/terrors"
	"golang.org/x/net/context"
)

var (
	// Client is used to send all requests by default. It can be overridden globally but MUST only be done before use
	// takes place; access is not synchronised.
	Client              = BareClient
	httpClientTransport = &httpcontrol.Transport{
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

// httpCancellationFilter ties together the context cancellation and the cancel channel of net/http.Request. It is
// incorporated into BareClient by default.
// @TODO: Go 1.7's http library has native context support, so this can go away
func httpCancellationFilter(req Request, svc Service) Response {
	ctx, ctxCancel := context.WithCancel(req.Context)
	defer ctxCancel()

	// When the context is cancelled, propagate this to net/http
	// If the caller set the net/http Cancel channel, allow this to be used too
	httpCancel := make(chan struct{})
	httpSuperCancel := req.Request.Cancel
	req.Request.Cancel = httpCancel
	go func() {
		select {
		case <-ctx.Done():
			close(httpCancel)
		case <-httpSuperCancel:
			close(httpCancel)
		case <-httpCancel:
		}
	}()

	req.Context = ctx
	return svc(req)
}

func BareClient(req Request) Response {
	return httpCancellationFilter(req, func(req Request) Response {
		httpRsp, err := httpClient.Do(&req.Request)

		// Read the response in its entirety and close the Response body here; this protects us from callers that forget to
		// call Close() but does not allow streaming responses.
		// @TODO: Streaming client?
		if httpRsp != nil && httpRsp.Body != nil {
			buf, err := ioutil.ReadAll(httpRsp.Body)
			httpRsp.Body.Close()
			if err != nil {
				log.Warn(req, "Error reading response body: %v", err)
			}
			httpRsp.Body = ioutil.NopCloser(bytes.NewReader(buf))
		}

		return Response{
			Response: httpRsp,
			Error:    terrors.Wrap(err, nil)}
	})
}

func SendVia(req Request, svc Service) *ResponseFuture {
	ctx, cancel := context.WithCancel(req.Context)
	req.Context = ctx
	f := &ResponseFuture{
		done:   ctx.Done(),
		cancel: cancel}
	go func() {
		defer cancel() // if already cancelled on escape, this is a no-op
		rsp := svc(req)
		f.mtx.RLock()
		f.r = rsp
		f.mtx.RUnlock()
	}()
	return f
}

func Send(req Request) *ResponseFuture {
	return SendVia(req, Client)
}
