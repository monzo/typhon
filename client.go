package typhon

import (
	"net/http"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/monzo/terrors"
)

var (
	// Client is used to send all requests by default. It can be overridden globally but MUST only be done before use
	// takes place; access is not synchronised.
	Client Service = BareClient
	// RoundTripper is used by default in Typhon clients
	RoundTripper http.RoundTripper = &httpcontrol.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		MaxIdleConnsPerHost: 10,
		DialKeepAlive:       10 * time.Minute,
		MaxTries:            6}
)

// A ResponseFuture is a container for a Response which will materialise at some point.
type ResponseFuture struct {
	done <-chan struct{} // guards access to r
	r    Response
}

// WaitC returns a channel which can be waited upon until the response is available
func (f *ResponseFuture) WaitC() <-chan struct{} {
	return f.done
}

// Response provides access to the response object, blocking until it is available
func (f *ResponseFuture) Response() Response {
	<-f.WaitC()
	return f.r
}

// HttpService returns a Service which sends requests via the given net/http RoundTripper.
// Only use this if you need to do something custom at the transport level.
func HttpService(rt http.RoundTripper) Service {
	return Service(func(req Request) Response {
		ctx := req.unwrappedContext()
		httpRsp, err := rt.RoundTrip(req.Request.WithContext(ctx))
		// When the calling context is cancelled, close the response body
		// This protects callers that forget to call Close(), or those which proxy responses upstream
		if httpRsp != nil && httpRsp.Body != nil {
			body := newDoneReader(httpRsp.Body)
			httpRsp.Body = body
			go func() {
				select {
				case <-body.done:
				case <-req.Done():
				}
				body.Close()
			}()
		}
		return Response{
			Response: httpRsp,
			Error:    terrors.Wrap(err, nil)}
	})
}

// BareClient is the most basic way to send a request, using the default http RoundTripper
func BareClient(req Request) Response {
	return HttpService(RoundTripper)(req)
}

// SendVia sends the given request via the given service, returning a future representing the operation
func SendVia(req Request, svc Service) *ResponseFuture {
	done := make(chan struct{}, 0)
	f := &ResponseFuture{
		done: done}
	go func() {
		defer close(done)
		f.r = svc(req)
	}()
	return f
}

// Send is equivalent to SendVia(req, Client)
func Send(req Request) *ResponseFuture {
	return SendVia(req, Client)
}
