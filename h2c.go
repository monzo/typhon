package typhon

import (
	"net/textproto"

	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// H2cFilter adds HTTP/2 h2c upgrade support to the wrapped Service (as defined in RFC 7540 Sections 3.2, 3.4).
func H2cFilter(req Request, svc Service) Response {
	h := req.Header
	// h2c with prior knowledge (RFC 7540 Section 3.4)
	isPrior := (req.Method == "PRI" && len(h) == 0 && req.URL.Path == "*" && req.Proto == "HTTP/2.0")
	// h2c upgrade (RFC 7540 Section 3.2)
	isUpgrade := httpguts.HeaderValuesContainsToken(h[textproto.CanonicalMIMEHeaderKey("Upgrade")], "h2c") &&
		httpguts.HeaderValuesContainsToken(h[textproto.CanonicalMIMEHeaderKey("Connection")], "HTTP2-Settings")
	if isPrior || isUpgrade {
		rsp := NewResponse(req)
		h2s := &http2.Server{}
		h2c.NewHandler(HttpHandler(svc), h2s).ServeHTTP(rsp.Writer(), &req.Request)
		return rsp
	}
	return svc(req)
}
