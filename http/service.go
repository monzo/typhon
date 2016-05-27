package httpsvc

// A Service is a function that takes a request, and produces a response. Services are used symetrically to
// represent both clients and servers.
type Service func(req Request) Response

// Filtered vends a new, filtered service.
func (svc Service) Filtered(f Filter) Service {
	return func(req Request) Response {
		return f(req, svc)
	}
}
