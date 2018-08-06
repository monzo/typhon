package typhon

// A Service is a function that takes a request and produces a response. Services are used symmetrically in
// both clients and servers.
type Service func(req Request) Response

// Filter vends a new service wrapped in the passed filter.
func (svc Service) Filter(f Filter) Service {
	return func(req Request) Response {
		return f(req, svc)
	}
}
