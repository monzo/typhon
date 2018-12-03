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
