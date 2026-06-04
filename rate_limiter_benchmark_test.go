package limen

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func newBenchmarkRateLimiter(b *testing.B, maxReqs int, window time.Duration) *rateLimiter {
	b.Helper()
	l, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: newTestMemoryAdapter(b),
		Secret:   testSecret,
	})
	if err != nil {
		b.Fatalf("New: %v", err)
	}
	return &rateLimiter{
		config: NewDefaultRateLimiterConfig(
			WithRateLimiterMaxRequests(maxReqs),
			WithRateLimiterWindow(window),
		),
		store: newRateLimiterCacheStore(l.core),
	}
}

func BenchmarkRateLimiterCheckCacheSameKey(b *testing.B) {
	rl := newBenchmarkRateLimiter(b, 1_000_000_000, time.Hour)
	rule := NewRateLimitRule("", 1_000_000_000, time.Hour)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := rl.Check(ctx, "same-key", rule); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRateLimiterCheckCacheNewKeys(b *testing.B) {
	rl := newBenchmarkRateLimiter(b, 100, time.Hour)
	rule := NewRateLimitRule("", 100, time.Hour)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if _, err := rl.Check(ctx, strconv.Itoa(i), rule); err != nil {
			b.Fatal(err)
		}
	}
}
