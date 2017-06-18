package typhon

import (
	"fmt"
	"sync"

	"github.com/labstack/echo"
	"github.com/monzo/terrors"
)

type Router interface {
	// OPTIONS is a shortcut for Register("OPTIONS", path, svc).
	OPTIONS(pattern string, svc Service)
	// GET is a shortcut for Register("GET", path, svc).
	GET(pattern string, svc Service)
	// HEAD is a shortcut for Register("HEAD", path, svc).
	HEAD(pattern string, svc Service)
	// POST is a shortcut for Register("POST", path, svc).
	POST(pattern string, svc Service)
	// PUT is a shortcut for Register("PUT", path, svc).
	PUT(pattern string, svc Service)
	// DELETE is a shortcut for Register("DELETE", path, svc).
	DELETE(pattern string, svc Service)
	// TRACE is a shortcut for Register("TRACE", path, svc).
	TRACE(pattern string, svc Service)
	// Register associates a Service with a method and path.
	Register(method, pattern string, svc Service)
	// Lookup returns the Service, pattern, and extracted path parameters for the HTTP method and path.
	Lookup(method, path string) (svc Service, pattern string, params map[string]string, ok bool)
	// Serve returns a Service which will route inbound requests to the enclosed routes.
	Serve() Service
	// Pattern returns the registered pattern which matches the given request.
	Pattern(req Request) string
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

func (r *router) Register(method, pattern string, svc Service) {
	r.m.Lock()
	defer r.m.Unlock()
	r.r.Add(method, pattern, func(c echo.Context) error { return nil })
	r.svcs[method+pattern] = svc
}

func (r *router) Lookup(method, path string) (Service, string, map[string]string, bool) {
	c := r.e.AcquireContext()
	defer r.e.ReleaseContext(c)
	c.Reset(nil, nil)
	c.SetPath("") // Annoyingly, this isn't done as part of Reset()

	r.m.RLock()
	r.r.Find(method, path, c)
	pattern := c.Path()
	if pattern == "" {
		r.m.RUnlock()
		return nil, "", nil, false
	}
	svc := r.svcs[method+pattern]
	r.m.RUnlock()

	if svc == nil {
		return nil, "", nil, false
	}

	names := c.ParamNames()
	params := make(map[string]string, len(names))
	for _, name := range names {
		params[name] = c.Param(name)
	}
	return svc, pattern, params, true
}

func (r *router) Serve() Service {
	return func(req Request) Response {
		svc, _, _, ok := r.Lookup(req.Method, req.URL.Path)
		if !ok {
			txt := fmt.Sprintf("No handler for %s %s", req.Method, req.URL.Path)
			rsp := NewResponse(req)
			rsp.Error = terrors.NotFound("no_handler", txt, nil)
			return rsp
		}
		return svc(req)
	}
}

func (r *router) Pattern(req Request) string {
	_, pattern, _, _ := r.Lookup(req.Method, req.URL.Path)
	return pattern
}

func (r *router) Params(req Request) map[string]string {
	_, _, params, _ := r.Lookup(req.Method, req.URL.Path)
	return params
}

// Sugar
func (r *router) OPTIONS(pattern string, svc Service) { r.Register("OPTIONS", pattern, svc) }
func (r *router) GET(pattern string, svc Service)     { r.Register("GET", pattern, svc) }
func (r *router) HEAD(pattern string, svc Service)    { r.Register("HEAD", pattern, svc) }
func (r *router) POST(pattern string, svc Service)    { r.Register("POST", pattern, svc) }
func (r *router) PUT(pattern string, svc Service)     { r.Register("PUT", pattern, svc) }
func (r *router) DELETE(pattern string, svc Service)  { r.Register("DELETE", pattern, svc) }
func (r *router) TRACE(pattern string, svc Service)   { r.Register("TRACE", pattern, svc) }
