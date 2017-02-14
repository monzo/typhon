package typhon

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/facebookgo/httpcontrol"
	log "github.com/monzo/slog"
	"github.com/monzo/terrors"
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

// HttpService returns a Service which sends requests via the given net/http RoundTripper.
// Only use this if you need to do something custom at the transport level.
func HttpService(rt http.RoundTripper) Service {
	return Service(func(req Request) Response {
		httpRsp, err := rt.RoundTrip(req.Request.WithContext(req.Context))
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
	})
}

func BareClient(req Request) Response {
	return HttpService(httpClientTransport)(req)
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
