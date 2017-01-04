package typhon

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/facebookgo/httpcontrol"
	log "github.com/monzo/slog"
	"github.com/monzo/terrors"
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
	httpClient = &http.Client{
		Timeout:   time.Hour,
		Transport: httpClientTransport}
)

type ResponseFuture struct {
	cancel context.CancelFunc
	done   <-chan struct{} // guards access to r
	r      Response
}

func (f *ResponseFuture) WaitC() <-chan struct{} {
	return f.done
}

func (f *ResponseFuture) Response() Response {
	<-f.WaitC()
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

// NewHttpClient returns a Service which sends requests via the given net/http client.
// You should not need to use this very often at all.
func NewHttpClient(c *http.Client) Service {
	return Service(func(req Request) Response {
		httpRsp, err := c.Do(&req.Request)
		// Read the response in its entirety and close the Response body here; this protects us from callers that forget to
		// call Close() but does not allow streaming responses.
		// @TODO: Streaming client?
		if httpRsp != nil && httpRsp.Body != nil {
			var buf []byte
			buf, err = ioutil.ReadAll(httpRsp.Body)
			httpRsp.Body.Close()
			if err != nil {
				log.Warn(req, "Error reading response body: %v", err)
			} else {
				httpRsp.Body = ioutil.NopCloser(bytes.NewReader(buf))
			}
		}

		return Response{
			Response: httpRsp,
			Error:    terrors.Wrap(err, nil)}
	}).Filter(httpCancellationFilter)
}

func BareClient(req Request) Response {
	return NewHttpClient(httpClient)(req)
}

func SendVia(req Request, svc Service) *ResponseFuture {
	ctx, cancel := context.WithCancel(req.Context)
	req.Context = ctx
	done := make(chan struct{}, 0)
	f := &ResponseFuture{
		done:   done,
		cancel: cancel}
	go func() {
		defer close(done)
		defer cancel() // if already cancelled on escape, this is a no-op
		f.r = svc(req)
	}()
	return f
}

func Send(req Request) *ResponseFuture {
	return SendVia(req, Client)
}
