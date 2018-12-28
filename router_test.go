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
	router := Router{}
	rsp := NewResponse(Request{})
	svc := func(req Request) Response {
		// For accuracy of the benchmarks it's important that this function perform no allocations. That's why we
		// construct the response outside the function
		return rsp
	}
	router.GET("/foo", svc)
	router.PUT("/foo/:param/:param2", svc)
	router.GET("/foo/:param/:param2", svc)
	router.GET("/foo/:param/baz", svc) // Should take precedence over the above
	router.GET("/anon-residual/*", svc)
	router.GET("/named-residual/*residual", svc)
	router.Register("*", "/poly", svc)

	cases := []routerTestCase{
		{
			// Unknown path: 404
			method: http.MethodGet,
			path:   "/",
			status: http.StatusNotFound,
		},
		{
			// Vanilla
			method:  http.MethodGet,
			path:    "/foo",
			status:  http.StatusOK,
			pattern: "/foo",
			params:  map[string]string{},
		},
		{
			// Params
			method:  http.MethodGet,
			path:    "/foo/bar2bär/baz",
			status:  http.StatusOK,
			pattern: "/foo/:param/baz",
			params: map[string]string{
				"param": "bar2bär"},
		},
		{
			method:  http.MethodPut,
			path:    "/foo/bar2bär/baz",
			status:  http.StatusOK,
			pattern: "/foo/:param/:param2",
			params: map[string]string{
				"param":  "bar2bär",
				"param2": "baz"},
		},
		{
			// Too many params
			method: http.MethodGet,
			path:   "/foo/bar/bar/baz",
			status: http.StatusNotFound,
		},
		{
			// Residual
			method:  http.MethodGet,
			path:    "/anon-residual/r",
			status:  http.StatusOK,
			pattern: "/anon-residual/*",
			params:  map[string]string{},
		},
		{
			// Longer residual
			method:  http.MethodGet,
			path:    "/anon-residual/r/e/s/i/d/u/a/l",
			status:  http.StatusOK,
			pattern: "/anon-residual/*",
			params:  map[string]string{},
		},
		{
			// Longer residual, trailing slash
			method:  http.MethodGet,
			path:    "/anon-residual/r/e/s/i/d/u/a/l/",
			status:  http.StatusOK,
			pattern: "/anon-residual/*",
			params:  map[string]string{},
		},
		{
			method:  http.MethodGet,
			path:    "/named-residual/r",
			status:  http.StatusOK,
			pattern: "/named-residual/*residual",
			params: map[string]string{
				"residual": "r"},
		},
		{
			// Esoteric poly-method
			method:  "WTAF",
			path:    "/poly",
			status:  http.StatusOK,
			pattern: "/poly",
			params:  map[string]string{},
		}}

	// Add a case per-method for the poly-method route
	for _, m := range [...]string{"GET", "CONNECT", "DELETE", "HEAD", "OPTIONS", "PATCH", "POST", "PUT", "TRACE"} {
		cases = append(cases, routerTestCase{
			method:  m,
			path:    "/poly",
			status:  http.StatusOK,
			pattern: "/poly",
			params:  map[string]string{},
		})
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
