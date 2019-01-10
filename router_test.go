package typhon

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type routerTestCase struct {
	// inputs

	method string
	path   string

	// expected outputs

	status  int
	pattern string
	params  map[string]string
}

func routerTestHarness() (Router, []routerTestCase) {
	rsp := NewResponse(Request{})
	svc := func(req Request) Response {
		// For accuracy of the benchmarks it's important that this function perform no allocations. That's why we
		// construct the response outside the function
		return rsp
	}

	router := Router{}
	router.GET("/", svc)
	router.GET("/index.html", svc)
	router.PUT("/:p/:p2", svc)
	router.Register("*", "/:p3/:p4", svc) // should take precedence over /put/:p/:p2
	router.GET("/:p5/:p6", svc)           // should take precedence over /poly/:p/:p2
	router.GET("/anon-residual/*", svc)
	router.GET("/residual/*r", svc)
	router.GET("/residual/*r/:p/:p2", svc)
	router.GET("/residual/*r/:p/:p2/*rr", svc)

	cases := []routerTestCase{
		// static
		{"GET", "/", 200, "/", map[string]string{}},
		{"GET", "/index.html", 200, "/index.html", map[string]string{}},
		// parameter extraction and precedence
		{"POST", "/1/2", 200, "/:p3/:p4", map[string]string{"p3": "1", "p4": "2"}},
		{"PUT", "/1/2", 200, "/:p3/:p4", map[string]string{"p3": "1", "p4": "2"}},
		{"GET", "/1/2", 200, "/:p5/:p6", map[string]string{"p5": "1", "p6": "2"}},
		{"GET", "/龍/Дракон", 200, "/:p5/:p6", map[string]string{"p5": "龍", "p6": "Дракон"}},
		{"WTAF", "/1/2", 200, "/:p3/:p4", map[string]string{"p3": "1", "p4": "2"}}, // * should match _any_ method
		// residuals
		{"GET", "/anon-residual/foo", 200, "/anon-residual/*", map[string]string{}},
		{"GET", "/anon-residual/foo/bar", 200, "/anon-residual/*", map[string]string{}},
		{"GET", "/anon-residual/foo/bar/", 200, "/anon-residual/*", map[string]string{}},
		{"GET", "/residual/foo", 200, "/residual/*r", map[string]string{"r": "foo"}},
		{"GET", "/residual/foo/bar", 200, "/residual/*r", map[string]string{"r": "foo/bar"}},
		{"GET", "/residual/foo/bar/", 200, "/residual/*r", map[string]string{"r": "foo/bar/"}},
		// complex combinations of residuals and named parameters
		{"GET", "/residual/foo/bar/baz", 200, "/residual/*r/:p/:p2", map[string]string{"r": "foo", "p": "bar", "p2": "baz"}},
		{"GET", "/residual/foo/bar/baz/bar/baz", 200, "/residual/*r/:p/:p2/*rr", map[string]string{"r": "foo", "p": "bar", "p2": "baz", "rr": "bar/baz"}},
		// not found
		{"GET", "/404", 404, "", map[string]string{}},
		{"GET", "/1/2/", 404, "", map[string]string{}}, // pattern doesn't include trailing slash
	}

	return router, cases
}

func TestRouter(t *testing.T) {
	t.Parallel()

	router, cases := routerTestHarness()
	svc := router.Serve().Filter(ErrorFilter)

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s%s", c.method, c.path), func(t *testing.T) {
			ctx := context.Background()
			req := NewRequest(ctx, c.method, c.path, nil)
			rsp := req.SendVia(svc).Response()

			assert.Equal(t, c.status, rsp.StatusCode)
			if rsp.StatusCode == http.StatusOK {
				require.NoError(t, rsp.Error)
				_, pattern, params, ok := router.Lookup(c.method, c.path)
				require.True(t, ok)
				assert.Equal(t, c.pattern, pattern)
				assert.Equal(t, c.params, params)
			}
		})
	}
}

func TestRouterForRequest(t *testing.T) {
	t.Parallel()

	router := Router{}
	var reqRouter *Router
	router.GET("/", func(req Request) Response {
		reqRouter = RouterForRequest(req)
		return req.Response(nil)
	})

	ctx := context.Background()
	router.Serve()(NewRequest(ctx, "GET", "/", nil))
	require.NotNil(t, reqRouter)
	assert.Equal(t, router, *reqRouter)
}

func TestRouterSetsRequest(t *testing.T) {
	t.Parallel()

	router := Router{}
	router.GET("/", func(req Request) Response {
		return Response{}
	})

	ctx := context.Background()
	req := NewRequest(ctx, "GET", "/", map[string]string{"r": "foo"})
	rsp := router.Serve()(req)
	require.NotNil(t, rsp.Request)
	// Request should be equal, bar the Context, which will have added value for routerContextKey
	req.Context = rsp.Request.Context
	assert.Equal(t, req, *rsp.Request)
}
