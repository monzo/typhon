package typhon

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var routeMatcherPatterns = []string{
	"",
	"/",
	"/index.html",
	"/foo",
	"/foo/",
	"/foo//bar",
	"/foo/bar.json",
	"/a+b/(c)/[d]",
	"/:p",
	"/:p/",
	":p",
	":p/x",
	"/:p/:p2",
	"/:p/:p",
	"/foo/:id",
	"/foo/:id/bar",
	"/foo/:id/bar/:other",
	"/龍/:p",
	"/foo:bar",
	"/foo/a:b",
	"/foo/:id.json",
	"/foo/:id:other",
	"/foo/:id-:other",
	"*",
	"*r",
	"/*",
	"/*r",
	"/foo/*",
	"/foo/*r",
	"/foo/:id/*r",
	"/foo/*r/bar",
	"/foo/*r/:p/:p2",
	"/foo/*r/:p/:p2/*rr",
	"/foo/*.json",
	"/foo*",
	"/\ufffd/:p",
	"/\xff/:p",
}

var routeMatcherPaths = []string{
	"",
	"/",
	"//",
	"/index.html",
	"/indexXhtml",
	"/foo",
	"/foo/",
	"/foo//",
	"/foo//bar",
	"/foo/bar",
	"/foo/bar/",
	"/foo/bar.json",
	"/foo/barXjson",
	"/foo/bar/baz",
	"/foo/bar/baz/",
	"/foo/bar/baz/qux",
	"/foo/bar/baz/qux/quux",
	"/foo/1/bar",
	"/foo/1/bar/2",
	"/foo/1/bar/2/3",
	"/foo:bar",
	"/foo/a:b",
	"/foo/1.json",
	"/foo/1:2",
	"/foo/1-2",
	"/a+b/(c)/[d]",
	"/龍/Дракон",
	"/foo/new\nline",
	"/foo/new\nline/x",
	"/foo/\n",
	"/\ufffd/x",
	"/\xff/x",
	"/foo/\xff",
	"foo",
	"foo/x",
	"/foo/x/y/z.json",
}

// assertMatcherAgreesWithRegexp checks that a routeMatcher (if one is built for the pattern) gives exactly the same
// answer and the same captured parameters as the regular expression the router compiles for it.
func assertMatcherAgreesWithRegexp(t testing.TB, pattern, path string) {
	t.Helper()
	m := newRouteMatcher(pattern)
	if m == nil {
		return
	}
	re := (&Router{}).compile(pattern)

	wantMatch := re.MatchString(path)
	wantParams := map[string]string{}
	if wantMatch && re.NumSubexp() > 0 {
		names := re.SubexpNames()[1:]
		for i, v := range re.FindStringSubmatch(path)[1:] {
			wantParams[names[i]] = v
		}
	}

	gotParams := map[string]string{}
	gotMatch := m.match(path, gotParams)
	if gotMatch != wantMatch {
		t.Fatalf("pattern %q path %q: matcher returned %v, regexp returned %v", pattern, path, gotMatch, wantMatch)
	}
	if gotMatch {
		assert.Equal(t, wantParams, gotParams, "pattern %q path %q", pattern, path)
	} else {
		assert.Empty(t, gotParams, "params must not be written on a failed match (pattern %q path %q)", pattern, path)
	}
	if gotMatch != m.match(path, nil) {
		t.Fatalf("pattern %q path %q: result depends on whether params are requested", pattern, path)
	}
}

func TestRouteMatcherAgreesWithRegexp(t *testing.T) {
	t.Parallel()
	for _, pattern := range routeMatcherPatterns {
		for _, path := range routeMatcherPaths {
			assertMatcherAgreesWithRegexp(t, pattern, path)
		}
	}
}

func TestRouteMatcherCoverage(t *testing.T) {
	t.Parallel()
	// Patterns that should take the fast path
	for _, pattern := range []string{
		"", "/", "/index.html", "/foo/:id", "/foo/:id/bar/:other", ":p", "/foo/*", "/foo/*r", "/foo/:id/*r", "*",
		"/foo:bar", "/foo/a:b",
	} {
		assert.NotNil(t, newRouteMatcher(pattern), "expected a matcher for %q", pattern)
	}
	// Patterns that need backtracking (or special UTF-8 handling) and must keep using the regexp
	for _, pattern := range []string{
		"/foo/:id.json", "/foo/:id-:other", "/foo/*r/bar", "/foo/*r/:p/:p2", "/foo/*.json", "/\ufffd/:p", "/\xff/:p",
	} {
		assert.Nil(t, newRouteMatcher(pattern), "expected no matcher for %q", pattern)
	}
}

func TestRouterMixedMatchersPrecedence(t *testing.T) {
	t.Parallel()
	// Fast-path and regexp entries interleaved: the last registered route must still win.
	router := Router{}
	svc := func(req Request) Response { return Response{} }
	router.GET("/files/:name", svc)        // fast path
	router.GET("/files/:name.json", svc)   // regexp
	router.GET("/files/*rest", svc)        // fast path, takes precedence over both
	router.GET("/files/*rest/meta", svc)   // regexp, takes precedence over everything for .../meta
	router.POST("/files/:name/meta", svc)  // different method
	router.Register("*", "/any/:x", svc)   // any method
	router.GET("/any/:x.txt", svc)         // GET-only, regexp
	router.Register("*", "/any/:x/y", svc) // any method, fast path

	cases := []struct {
		method, path, pattern string
		params                map[string]string
	}{
		{"GET", "/files/a", "/files/*rest", map[string]string{"rest": "a"}},
		{"GET", "/files/a.json", "/files/*rest", map[string]string{"rest": "a.json"}},
		{"GET", "/files/a/b/meta", "/files/*rest/meta", map[string]string{"rest": "a/b"}},
		{"POST", "/files/a/meta", "/files/:name/meta", map[string]string{"name": "a"}},
		{"get", "/any/a.txt", "/any/:x.txt", map[string]string{"x": "a"}},
		{"DELETE", "/any/a.txt", "/any/:x", map[string]string{"x": "a.txt"}},
		{"PATCH", "/any/a/y", "/any/:x/y", map[string]string{"x": "a"}},
	}
	for _, c := range cases {
		_, pattern, params, ok := router.Lookup(c.method, c.path)
		require.True(t, ok, "%s %s", c.method, c.path)
		assert.Equal(t, c.pattern, pattern, "%s %s", c.method, c.path)
		assert.Equal(t, c.params, params, "%s %s", c.method, c.path)
	}

	_, _, _, ok := router.Lookup("POST", "/files/a")
	assert.False(t, ok)
}

func FuzzRouteMatcherAgreesWithRegexp(f *testing.F) {
	for _, pattern := range routeMatcherPatterns {
		for _, path := range routeMatcherPaths {
			f.Add(pattern, path)
		}
	}
	f.Fuzz(func(t *testing.T, pattern, path string) {
		assertMatcherAgreesWithRegexp(t, pattern, path)
	})
}

type routedContextTestKey struct{}

func TestRoutedContext(t *testing.T) {
	t.Parallel()

	parent, cancel := context.WithCancel(context.WithValue(context.Background(), routedContextTestKey{}, "parent value"))
	router := &Router{}
	ctx := newRoutedContext(parent, router, "/foo/:id", "GET")

	assert.Equal(t, router, ctx.Value(routerContextKey))
	assert.Equal(t, "/foo/:id", ctx.Value(routerRequestPatternContextKey))
	assert.Equal(t, "GET", ctx.Value(routerRequestMethodContextKey))
	assert.Equal(t, "parent value", ctx.Value(routedContextTestKey{}))
	assert.Nil(t, ctx.Value("missing"))

	pattern, ok := RequestPatternFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "/foo/:id", pattern)
	method, ok := RequestMethodFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "GET", method)

	// Values set further down the chain still shadow the router's ones
	child := context.WithValue(ctx, routerRequestPatternContextKey, "/override")
	pattern, _ = RequestPatternFromContext(child)
	assert.Equal(t, "/override", pattern)

	// Cancellation of the parent propagates through to contexts derived from the routed context
	derived, cancelDerived := context.WithCancel(ctx)
	defer cancelDerived()
	cancel()
	select {
	case <-derived.Done():
	case <-time.After(time.Second):
		t.Fatal("cancellation did not propagate through routedContext")
	}
	assert.ErrorIs(t, derived.Err(), context.Canceled)

	assert.Panics(t, func() { newRoutedContext(nil, router, "/", "GET") })
}
