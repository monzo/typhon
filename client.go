package typhon

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/monzo/terrors"
)

var (
	// Client is used to send all requests by default. It can be overridden globally but MUST only be done before use
	// takes place; access is not synchronised.
	Client Service = BareClient
	// RoundTripper chooses HTTP1, or H2C based on a context flag (see WithH2C)
	RoundTripper http.RoundTripper = dynamicRoundTripper{}

	// HTTPRoundTripper is a HTTP1 and TLS HTTP2 client
	HTTPRoundTripper http.RoundTripper = &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		IdleConnTimeout:     10 * time.Minute,
		MaxIdleConnsPerHost: 10,
	}

	// H2cRoundTripper is a prior-knowledge H2c client. It does not support ProxyFromEnvironment.
	H2cRoundTripper http.RoundTripper = newH2cRoundTripper()
)

// newH2cRoundTripper builds a transport that speaks unencrypted HTTP/2 (h2c) with prior knowledge
func newH2cRoundTripper() *http.Transport {
	t := &http.Transport{
		HTTP2: &http.HTTP2Config{
			// Health-check idle connections: send a ping after 30s without a frame, and close the
			// connection if no response arrives within 10s.
			SendPingTimeout: 30 * time.Second,
			PingTimeout:     10 * time.Second,
		},
	}
	// Enabling UnencryptedHTTP2 without HTTP1 makes the transport use h2c (prior knowledge) for
	// http:// URLs with no HTTP/1.1 fallback.
	t.Protocols = new(http.Protocols)
	t.Protocols.SetUnencryptedHTTP2(true)
	return t
}

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
	return func(req Request) Response {
		ctx := req.unwrappedContext()
		httpReq := req.Request.WithContext(ctx)
		// Expose GetBody for buffered bodies so the HTTP/2 transport can rewind and transparently
		// retry the request when a connection is torn down before the response arrives (eg. a
		// graceful-shutdown GOAWAY). Without GetBody the transport can't replay the body and the
		// request fails outright. Streaming bodies aren't a *bufCloser, so they stay non-replayable.
		if buf, ok := httpReq.Body.(*bufCloser); ok {
			body := buf.Bytes()
			httpReq.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(body)), nil
			}
		}
		httpRsp, err := rt.RoundTrip(httpReq)
		// When the calling context is cancelled, close the response body
		// This protects callers that forget to call Close(), or those which proxy responses upstream
		//
		// If the calling context is infinite (ie. returns nil for Done()), it can never signal cancellation
		// so we bypass this as a performance optimisation
		if httpRsp != nil && httpRsp.Body != nil && ctx.Done() != nil {
			body := newDoneReader(httpRsp.Body, httpRsp.ContentLength)
			httpRsp.Body = body
			go func() {
				select {
				case <-body.closed:
				case <-ctx.Done():
					body.Close()
				}
			}()
		}
		return Response{
			Request:  &req,
			Response: httpRsp,
			Error:    terrors.Wrap(err, nil)}
	}
}

// BareClient is the most basic way to send a request, using the default http RoundTripper
func BareClient(req Request) Response {
	return HttpService(RoundTripper)(req)
}

// SendVia round-trips the request via the passed Service. It does not block, instead returning a ResponseFuture
// representing the asynchronous operation to produce the response.
func SendVia(req Request, svc Service) *ResponseFuture {
	done := make(chan struct{}, 0)
	f := &ResponseFuture{
		done: done}
	go func() {
		defer close(done) // makes the response available to waiters
		f.r = svc(req)
	}()
	return f
}

// Send round-trips the request via the default Client. It does not block, instead returning a ResponseFuture
// representing the asynchronous operation to produce the response. It is equivalent to:
//
//	SendVia(req, Client)
func Send(req Request) *ResponseFuture {
	return SendVia(req, Client)
}

type withH2C struct{}

// WithH2C instructs the dynamicRoundTripper to use prior-knowledge cleartext HTTP2 instead of HTTP1.1
func WithH2C(ctx context.Context) context.Context {
	return context.WithValue(ctx, withH2C{}, true)
}

func isH2C(ctx context.Context) bool {
	b, _ := ctx.Value(withH2C{}).(bool)
	return b
}

type dynamicRoundTripper struct{}

func (d dynamicRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Scheme == "http" && isH2C(r.Context()) {
		return H2cRoundTripper.RoundTrip(r)
	}
	return HTTPRoundTripper.RoundTrip(r)
}
