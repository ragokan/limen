package limen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newBenchmarkRouter(middleware ...Middleware) *router {
	r := newRouter(nil, middleware...)
	r.AddRoute(MethodGET, "/session", benchmarkNoopHandler, "session", nil)
	r.AddRoute(MethodGET, "/users/:id", benchmarkNoopHandler, "user", nil)
	return r
}

func benchmarkNoopHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func benchmarkRouterRequest(b *testing.B, r *router, method, target string) {
	b.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, target, http.NoBody)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkRouterStaticRoute(b *testing.B) {
	benchmarkRouterRequest(b, newBenchmarkRouter(), http.MethodGet, "/session")
}

func BenchmarkRouterParamRoute(b *testing.B) {
	benchmarkRouterRequest(b, newBenchmarkRouter(), http.MethodGet, "/users/123")
}

func BenchmarkRouterWithMiddleware(b *testing.B) {
	middleware := Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})
	nextMiddleware := Middleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})
	benchmarkRouterRequest(b, newBenchmarkRouter(middleware, nextMiddleware), http.MethodGet, "/users/123")
}
