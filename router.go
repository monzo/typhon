package typhon

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mondough/terrors"
)

type Router interface {
	// OPTIONS is a shortcut for Register("PUT", svc).
	OPTIONS(path string, svc Service)
	// GET is a shortcut for Register("GET", svc).
	GET(path string, svc Service)
	// HEAD is a shortcut for Register("HEAD", svc).
	HEAD(path string, svc Service)
	// POST is a shortcut for Register("Delete", svc).
	POST(path string, svc Service)
	// PUT is a shortcut for Register("Delete", svc).
	PUT(path string, svc Service)
	// DELETE is a shortcut for Register("Delete", svc).
	DELETE(path string, svc Service)
	// TRACE is a shortcut for Register("Delete", svc).
	TRACE(path string, svc Service)
	// Register associates a Service with a method and path.
	Register(method, path string, svc Service)
	// Lookup returns the Service and extracted path parameters for the HTTP method and path.
	Lookup(method, path string) (svc Service, params map[string]string, ok bool)
	// Serve returns a Service which will route inbound requests to the enclosed routes.
	Serve() Service
	// Params returns extracted URL parameters, assuming the request has been routed and has captured parameters.
	Params(req Request) map[string]string
}

type router struct {
	impl *httprouter.Router
}

func NewRouter() Router {
	return &router{
		impl: httprouter.New()}
}

func (r *router) Register(method, path string, svc Service) {
	// Forgive me.
	r.impl.Handle(method, path, func(rw_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		rw := rw_.(*routerRw)
		rw.svc = svc
	})
}

func (r *router) Lookup(method, path string) (Service, map[string]string, bool) {
	hf, params_, _ := r.impl.Lookup(method, path)
	if hf == nil {
		return nil, nil, false
	}

	params := make(map[string]string, len(params_))
	for _, p := range params_ {
		params[p.Key] = p.Value
	}

	rw := routerRw{}
	hf(&rw, nil, nil)
	return rw.svc, params, true
}

func (r *router) Serve() Service {
	return func(req Request) Response {
		svc, _, ok := r.Lookup(req.Method, req.URL.Path)
		if !ok {
			txt := fmt.Sprintf("No handler for %s %s", req.Method, req.URL.Path)
			rsp := NewResponse(req)
			rsp.Error = terrors.NotFound("no_handler", txt, nil)
			return rsp
		}
		return svc(req)
	}
}

func (r *router) Params(req Request) map[string]string {
	_, params, _ := r.Lookup(req.Method, req.URL.Path)
	return params
}

// Sugar
func (r *router) OPTIONS(path string, svc Service) { r.Register("OPTIONS", path, svc) }
func (r *router) GET(path string, svc Service)     { r.Register("GET", path, svc) }
func (r *router) HEAD(path string, svc Service)    { r.Register("HEAD", path, svc) }
func (r *router) POST(path string, svc Service)    { r.Register("POST", path, svc) }
func (r *router) PUT(path string, svc Service)     { r.Register("PUT", path, svc) }
func (r *router) DELETE(path string, svc Service)  { r.Register("DELETE", path, svc) }
func (r *router) TRACE(path string, svc Service)   { r.Register("TRACE", path, svc) }

// I'm sorry, dear reader, I really am. To do this properly is more work than I have the appetite for right now.
//
// Future me will remove this horrific cruft and provide a URL router that acts on Services directly, without needing
// the kabuki of a fake Handler and ResponseWriter.
//
// As it is, here's the fake ResponseWriter.
type routerRw struct {
	svc Service
}

func (r *routerRw) Header() http.Header         { return nil }
func (r *routerRw) WriteHeader(_ int)           {}
func (r *routerRw) Write(_ []byte) (int, error) { return 0, nil }
