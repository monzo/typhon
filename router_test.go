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
	router := NewRouter()
	svc := func(req Request) Response {
		rsp := NewResponse(req)
		rsp.Header.Set("Router-Pattern", router.Pattern(req))
		rsp.Encode(router.Params(req))
		return rsp
	}
	router.GET("/foo", svc)
	router.GET("/foo/:param/baz", svc)
	router.GET("/residual/*residuals", svc)
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
			// Too many params
			method: http.MethodGet,
			path:   "/foo/bar/bar/baz",
			status: http.StatusNotFound,
		},
		{
			// Residual
			method:  http.MethodGet,
			path:    "/residual/r",
			status:  http.StatusOK,
			pattern: "/residual/*residuals",
			params: map[string]string{
				"*": "r"},
		},
		{
			// Longer residual
			method:  http.MethodGet,
			path:    "/residual/r/e/s/i/d/u/a/l",
			status:  http.StatusOK,
			pattern: "/residual/*residuals",
			params: map[string]string{
				"*": "r/e/s/i/d/u/a/l"},
		},
		{
			// Longer residual, trailing slash
			method:  http.MethodGet,
			path:    "/residual/r/e/s/i/d/u/a/l/",
			status:  http.StatusOK,
			pattern: "/residual/*residuals",
			params: map[string]string{
				"*": "r/e/s/i/d/u/a/l/"},
		},
		{
			// Unknown poly-method
			method: "WTAF",
			path:   "/poly",
			status: http.StatusNotFound,
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

			assert.Equal(t, rsp.StatusCode, c.status)
			if rsp.StatusCode == http.StatusOK {
				require.NoError(t, rsp.Error)
				assert.Equal(t, c.pattern, rsp.Header.Get("Router-Pattern"))

				params := map[string]string{}
				require.NoError(t, rsp.Decode(&params))
				assert.Equal(t, c.params, params)
			}
		})
	}
}

func BenchmarkRouter(b *testing.B) {
	router, cases := routerTestHarness()

	// Lookup benchmarks
	for _, c := range cases {
		b.Run(fmt.Sprintf("Lookup/%s%s", c.method, c.path), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				router.Lookup(c.method, c.path)
			}
		})
	}

	// Serve benchmarks
	ctx := context.Background()
	svc := router.Serve()
	for _, c := range cases {
		b.Run(fmt.Sprintf("Serve/%s%s", c.method, c.path), func(b *testing.B) {
			b.ReportAllocs()
			req := NewRequest(ctx, c.method, c.path, nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				svc(req)
			}
		})
	}
}
