package typhon

import (
	"fmt"
	"sync"

	"github.com/labstack/echo"
	"github.com/monzo/terrors"
)

type Router interface {
	// OPTIONS is a shortcut for Register("OPTIONS", svc).
	OPTIONS(path string, svc Service)
	// GET is a shortcut for Register("GET", svc).
	GET(path string, svc Service)
	// HEAD is a shortcut for Register("HEAD", svc).
	HEAD(path string, svc Service)
	// POST is a shortcut for Register("POST", svc).
	POST(path string, svc Service)
	// PUT is a shortcut for Register("PUT", svc).
	PUT(path string, svc Service)
	// DELETE is a shortcut for Register("DELETE", svc).
	DELETE(path string, svc Service)
	// TRACE is a shortcut for Register("TRACE", svc).
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
	e    *echo.Echo
	r    *echo.Router
	svcs map[string]Service
	m    sync.RWMutex
}

// NewRouter vends a new implementation of Router
func NewRouter() Router {
	e := echo.New()
	return &router{
		e:    e,
		r:    echo.NewRouter(e),
		svcs: make(map[string]Service, 10)}
}

func (r *router) identityHandler(c echo.Context) error {
	return nil
}

func (r *router) Register(method, path string, svc Service) {
	r.m.Lock()
	defer r.m.Unlock()
	r.r.Add(method, path, r.identityHandler)
	r.svcs[method+path] = svc
}

func (r *router) Lookup(method, path string) (Service, map[string]string, bool) {
	c := r.e.AcquireContext()
	defer r.e.ReleaseContext(c)
	c.Reset(nil, nil)
	c.SetPath("") // Annoyingly, this isn't done as part of Reset()

	r.m.RLock()
	r.r.Find(method, path, c)
	if c.Path() == "" {
		r.m.RUnlock()
		return nil, nil, false
	}
	svc := r.svcs[method+c.Path()]
	r.m.RUnlock()

	if svc == nil {
		return nil, nil, false
	}

	names := c.ParamNames()
	params := make(map[string]string, len(names))
	for _, name := range names {
		params[name] = c.Param(name)
	}
	return svc, params, true

	// hf, params_, _ := r.impl.Lookup(method, path)
	// if hf == nil {
	// 	return nil, nil, false
	// }
	//
	// params := make(map[string]string, len(params_))
	// for _, p := range params_ {
	// 	params[p.Key] = p.Value
	// }
	//
	// rw := routerRw{}
	// hf(&rw, nil, nil)
	// return rw.svc, params, true
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
