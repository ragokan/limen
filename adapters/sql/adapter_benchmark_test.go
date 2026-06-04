package sql

import (
	"context"
	"fmt"
	"testing"

	"github.com/ragokan/limen"
)

func setupBenchmarkDB(b *testing.B, rows int) *Adapter {
	b.Helper()
	adapter := setupTestDB(b)
	ctx := context.Background()
	for i := range rows {
		if err := adapter.Create(ctx, "test_items", map[string]any{
			"name":  fmt.Sprintf("User %d", i),
			"email": fmt.Sprintf("user-%d@example.com", i),
			"age":   i,
		}); err != nil {
			b.Fatalf("seed item: %v", err)
		}
	}
	return adapter
}

func BenchmarkSQLAdapterFindOne(b *testing.B) {
	adapter := setupBenchmarkDB(b, 100)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := adapter.FindOne(ctx, "test_items", []limen.Where{
			limen.Eq("email", "user-42@example.com"),
		}, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLAdapterFindMany(b *testing.B) {
	adapter := setupBenchmarkDB(b, 100)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := adapter.FindMany(ctx, "test_items", nil, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLAdapterCreate(b *testing.B) {
	adapter := setupBenchmarkDB(b, 0)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if err := adapter.Create(ctx, "test_items", map[string]any{
			"name":  fmt.Sprintf("Bench %d", i),
			"email": fmt.Sprintf("bench-%d@example.com", i),
			"age":   i,
		}); err != nil {
			b.Fatal(err)
		}
	}
}
