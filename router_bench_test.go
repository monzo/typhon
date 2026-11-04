package typhon

import (
	"context"
	"fmt"
	"testing"
)

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

// BenchmarkRouterManyRoutes routes requests through a router shaped like a typical service, with a few dozen routes.
// Routes are tried in reverse registration order, so requests for routes registered early have to be checked against
// most of the table first.
func BenchmarkRouterManyRoutes(b *testing.B) {
	svc := func(req Request) Response { return Response{} }
	router := Router{}
	resources := []string{"accounts", "balances", "cards", "pots", "transactions", "merchants", "payees", "webhooks"}
	for _, res := range resources {
		router.GET("/"+res, svc)
		router.POST("/"+res, svc)
		router.GET("/"+res+"/:id", svc)
		router.PUT("/"+res+"/:id", svc)
		router.GET("/"+res+"/:id/history/*rest", svc)
	}

	cases := []struct{ name, method, path string }{
		{"first-registered", "GET", "/accounts/acc_123"},
		{"last-registered", "GET", "/webhooks/wh_123/history/2024/01"},
		{"not-found", "GET", "/nope/123"},
	}
	ctx := context.Background()
	serve := router.Serve()
	for _, c := range cases {
		b.Run("Serve/"+c.name, func(b *testing.B) {
			b.ReportAllocs()
			req := NewRequest(ctx, c.method, c.path, nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				serve(req)
			}
		})
		b.Run("Params/"+c.name, func(b *testing.B) {
			b.ReportAllocs()
			req := NewRequest(ctx, c.method, c.path, nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				router.Params(req)
			}
		})
	}
}
