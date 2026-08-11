package typhon

// H2cFilter previously added HTTP/2 h2c upgrade support to the wrapped Service by hijacking the
// connection and running golang.org/x/net/http2/h2c.
//
// Deprecated: h2c (HTTP/2 cleartext, prior knowledge) is now enabled natively for every Typhon
// server via Go's net/http support (Go 1.24+); see Serve. This filter is a no-op retained only for
// backwards compatibility and will be removed in a future release. It is no longer necessary to
// apply it. Note that the native implementation only supports h2c with prior knowledge, not the
// (unused by Go clients) RFC 7540 §3.2 `Upgrade: h2c` handshake.
func H2cFilter(req Request, svc Service) Response {
	return svc(req)
}
