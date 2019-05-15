package typhon

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/monzo/terrors"
)

// We use a custom type to guarantee we won't get a collision with another package. Using an anonymous struct type
// directly means we'd get a collision with any other package that does the same.
// https://play.golang.org/p/MxhRiL37R-9
type routerContextKeyType struct{}

var (
	routerContextKey   = routerContextKeyType{}
	routerComponentsRe = regexp.MustCompile(`(?:^|/)(\*\w*|:\w+)`)
)

type routerEntry struct {
	Method  string
	Pattern string
	Service Service
	re      *regexp.Regexp
}

func (e routerEntry) String() string {
	return fmt.Sprintf("%s %s", e.Method, e.Pattern)
}

// A Router multiplexes requests to a set of Services by pattern matching on method and path, and can also extract
// parameters from paths.
type Router struct {
	entries []routerEntry
}

// RouterForRequest returns a pointer to the Router that successfully dispatched the request, or nil.
func RouterForRequest(r Request) *Router {
	if v := r.Context.Value(routerContextKey); v != nil {
		return v.(*Router)
	}
	return nil
}

func (r *Router) compile(pattern string) *regexp.Regexp {
	re, pos := ``, 0
	for _, m := range routerComponentsRe.FindAllStringSubmatchIndex(pattern, -1) {
		re += regexp.QuoteMeta(pattern[pos:m[2]]) // head
		token := pattern[m[2]:m[3]]
		switch sigil, name := token[0], token[1:]; sigil {
		case '*':
			if len(name) == 0 { // bare residual (*): doesn't capture what it consumes
				re += `.*?`
			} else { // named residual (*name): captures what it consumes
				re += `(?P<` + name + `>.*?)`
			}
		case ':':
			re += `(?P<` + name + `>[^/]+)`
		default:
			panic(fmt.Errorf("unhandled router token %#v", token))
		}
		pos = m[3]
	}
	re += regexp.QuoteMeta(pattern[pos:]) // tail
	re = `^` + re + `$`
	return regexp.MustCompile(re)
}

// Register associates a Service with a method and path.
//
// Method is a HTTP method name, or "*" to match any method.
//
// Patterns are strings of the format: /foo/:name/baz/*residual
// As well as being literal paths, they can contain named parameters like :name whose value is dynamic and only known at
// runtime, or *residual components which match (potentially) multiple path components.
//
// In the case that patterns are ambiguous, the last route to be registered will take precedence.
func (r *Router) Register(method, pattern string, svc Service) {
	re := r.compile(pattern)
	r.entries = append(r.entries, routerEntry{
		Method:  strings.ToUpper(method),
		Pattern: pattern,
		Service: svc,
		re:      re})
}

// lookup is the internal version of Lookup, but it extracts path parameters into the passed map (and skips it if the
// map is nil)
func (r Router) lookup(method, path string, params map[string]string) (Service, string, bool) {
	method = strings.ToUpper(method)
	for i := len(r.entries) - 1; i >= 0; i-- { // iterate in reverse to prefer routes registered later
		e := r.entries[i]
		if (e.Method == method || e.Method == `*`) && e.re.MatchString(path) {
			// We have a match
			if params != nil && e.re.NumSubexp() > 0 { // extract params
				names := e.re.SubexpNames()[1:]
				for i, value := range e.re.FindStringSubmatch(path)[1:] {
					params[names[i]] = value
				}
			}
			return e.Service, e.Pattern, true
		}
	}
	return nil, "", false
}

// Lookup returns the Service, pattern, and extracted path parameters for the HTTP method and path.
func (r Router) Lookup(method, path string) (Service, string, map[string]string, bool) {
	params := map[string]string{}
	svc, pattern, ok := r.lookup(method, path, params)
	return svc, pattern, params, ok
}

// Serve returns a Service which will route inbound requests to the enclosed routes.
func (r Router) Serve() Service {
	return func(req Request) Response {
		svc, _, ok := r.lookup(req.Method, req.URL.Path, nil)
		if !ok {
			txt := fmt.Sprintf("No handler for %s %s", req.Method, req.URL.Path)
			rsp := NewResponse(req)
			rsp.Error = terrors.NotFound("no_handler", txt, nil)
			return rsp
		}
		req.Context = context.WithValue(req.Context, routerContextKey, &r)
		rsp := svc(req)
		if rsp.Request == nil {
			rsp.Request = &req
		}
		return rsp
	}
}

// Pattern returns the registered pattern which matches the given request.
func (r Router) Pattern(req Request) string {
	_, pattern, _ := r.lookup(req.Method, req.URL.Path, nil)
	return pattern
}

// Params returns extracted path parameters, assuming the request has been routed and has captured parameters.
func (r Router) Params(req Request) map[string]string {
	_, _, params, _ := r.Lookup(req.Method, req.URL.Path)
	return params
}

// Sugar

// GET is shorthand for:
//  r.Register("GET", pattern, svc)
func (r *Router) GET(pattern string, svc Service) { r.Register("GET", pattern, svc) }

// CONNECT is shorthand for:
//  r.Register("CONNECT", pattern, svc)
func (r *Router) CONNECT(pattern string, svc Service) { r.Register("CONNECT", pattern, svc) }

// DELETE is shorthand for:
//  r.Register("DELETE", pattern, svc)
func (r *Router) DELETE(pattern string, svc Service) { r.Register("DELETE", pattern, svc) }

// HEAD is shorthand for:
//  r.Register("HEAD", pattern, svc)
func (r *Router) HEAD(pattern string, svc Service) { r.Register("HEAD", pattern, svc) }

// OPTIONS is shorthand for:
//  r.Register("OPTIONS", pattern, svc)
func (r *Router) OPTIONS(pattern string, svc Service) { r.Register("OPTIONS", pattern, svc) }

// PATCH is shorthand for:
//  r.Register("PATCH", pattern, svc)
func (r *Router) PATCH(pattern string, svc Service) { r.Register("PATCH", pattern, svc) }

// POST is shorthand for:
//  r.Register("POST", pattern, svc)
func (r *Router) POST(pattern string, svc Service) { r.Register("POST", pattern, svc) }

// PUT is shorthand for:
//  r.Register("PUT", pattern, svc)
func (r *Router) PUT(pattern string, svc Service) { r.Register("PUT", pattern, svc) }

// TRACE is shorthand for:
//  r.Register("TRACE", pattern, svc)
func (r *Router) TRACE(pattern string, svc Service) { r.Register("TRACE", pattern, svc) }
