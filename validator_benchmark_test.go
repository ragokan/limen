package limen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkValidatorEmail(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		v := NewValidator()
		v.Email("email", "person@example.com")
		if err := v.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateJSONBodyLookup(b *testing.B) {
	req := requestWithBenchmarkJSONBody(map[string]any{"email": "person@example.com"})
	b.ReportAllocs()
	for b.Loop() {
		if got := GetJSONBody(req); got["email"] != "person@example.com" {
			b.Fatal(got)
		}
	}
}

func requestWithBenchmarkJSONBody(body map[string]any) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", http.NoBody)
	return req.WithContext(context.WithValue(req.Context(), bodyContextKey{}, body))
}
