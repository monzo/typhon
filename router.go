package typhon

import (
	"fmt"
	"sync"

	"github.com/labstack/echo"
	"github.com/monzo/terrors"
)

// A Router multiplexes requests to a set of Services by pattern matching on method and path, and can also extract
// parameters from paths.
type Router struct {
	e    *echo.Echo
	r    *echo.Router
	svcs map[string]Service
	m    *sync.RWMutex
}

// NewRouter vends a new implementation of Router
func NewRouter() Router {
	e := echo.New()
	return Router{
		e:    e,
		r:    echo.NewRouter(e),
		svcs: make(map[string]Service, 10),
		m:    new(sync.RWMutex)}
}

// Register associates a Service with a method and path.
//
// Method is a single HTTP method name, or * which is expanded to {OPTIONS, GET, HEAD, POST, PUT, DELETE, TRACE}.
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) Register(method, pattern string, svc Service) {
	echoHandler := func(c echo.Context) error { return nil }

	r.m.Lock()
	defer r.m.Unlock()

	if method == "*" {
		// Expand * to the set of all known methods
		for _, m := range [...]string{"GET", "CONNECT", "DELETE", "HEAD", "OPTIONS", "PATCH", "POST", "PUT", "TRACE"} {
			r.r.Add(m, pattern, echoHandler)
			r.svcs[m+pattern] = svc
		}
	} else {
		r.r.Add(method, pattern, echoHandler)
		r.svcs[method+pattern] = svc
	}
}

// lookup is the internal version of Lookup, but it extracts path parameters into the passed map (and skips it if the
// map is nil)
func (r *Router) lookup(method, path string, params map[string]string) (Service, string, bool) {
	c := r.e.AcquireContext()
	defer r.e.ReleaseContext(c)
	c.Reset(nil, nil)
	c.SetPath("") // Annoyingly, this isn't done as part of Reset()

	r.m.RLock()
	r.r.Find(method, path, c)
	pattern := c.Path()
	if pattern == "" {
		r.m.RUnlock()
		return nil, "", false
	}
	svc := r.svcs[method+pattern]
	r.m.RUnlock()

	if svc == nil {
		return nil, "", false
	}

	if params != nil {
		names := c.ParamNames()
		for _, name := range names {
			params[name] = c.Param(name)
		}
	}
	return svc, pattern, true
}

// Lookup returns the Service, pattern, and extracted path parameters for the HTTP method and path.
func (r *Router) Lookup(method, path string) (Service, string, map[string]string, bool) {
	params := map[string]string{}
	svc, pattern, ok := r.lookup(method, path, params)
	return svc, pattern, params, ok
}

// Serve returns a Service which will route inbound requests to the enclosed routes.
func (r *Router) Serve() Service {
	return func(req Request) Response {
		svc, _, ok := r.lookup(req.Method, req.URL.Path, nil)
		if !ok {
			txt := fmt.Sprintf("No handler for %s %s", req.Method, req.URL.Path)
			rsp := NewResponse(req)
			rsp.Error = terrors.NotFound("no_handler", txt, nil)
			return rsp
		}
		return svc(req)
	}
}

// Pattern returns the registered pattern which matches the given request.
func (r *Router) Pattern(req Request) string {
	_, pattern, _ := r.lookup(req.Method, req.URL.Path, nil)
	return pattern
}

// Params returns extracted path parameters, assuming the request has been routed and has captured parameters.
func (r *Router) Params(req Request) map[string]string {
	_, _, params, _ := r.Lookup(req.Method, req.URL.Path)
	return params
}

// Sugar

// GET is shorthand for Register("GET", pattern, svc).
//
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) GET(pattern string, svc Service) { r.Register("GET", pattern, svc) }

// CONNECT is shorthand for Register("CONNECT", pattern, svc).
//
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) CONNECT(pattern string, svc Service) { r.Register("CONNECT", pattern, svc) }

// DELETE is shorthand for Register("DELETE", pattern, svc).
//
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) DELETE(pattern string, svc Service) { r.Register("DELETE", pattern, svc) }

// HEAD is shorthand for Register("HEAD", pattern, svc).
//
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) HEAD(pattern string, svc Service) { r.Register("HEAD", pattern, svc) }

// OPTIONS is shorthand for Register("OPTIONS", pattern, svc).
//
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) OPTIONS(pattern string, svc Service) { r.Register("OPTIONS", pattern, svc) }

// PATCH is shorthand for Register("PATCH", pattern, svc).
//
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) PATCH(pattern string, svc Service) { r.Register("PATCH", pattern, svc) }

// POST is shorthand for Register("POST", pattern, svc).
//
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) POST(pattern string, svc Service) { r.Register("POST", pattern, svc) }

// PUT is shorthand for Register("PUT", pattern, svc).
//
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) PUT(pattern string, svc Service) { r.Register("PUT", pattern, svc) }

// TRACE is shorthand for Register("TRACE", pattern, svc).
//
// Pattern syntax is as described in echo's documentation: https://echo.labstack.com/guide/routing
func (r *Router) TRACE(pattern string, svc Service) { r.Register("TRACE", pattern, svc) }
