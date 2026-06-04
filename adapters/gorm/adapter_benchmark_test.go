package gorm

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ragokan/limen"
)

func setupBenchmarkGormDB(b *testing.B, rows int) *Adapter {
	b.Helper()
	adapter := setupTestGormDB(b)
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

func BenchmarkGormAdapterFindOne(b *testing.B) {
	adapter := setupBenchmarkGormDB(b, 100)
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

func BenchmarkGormAdapterFindMany(b *testing.B) {
	adapter := setupBenchmarkGormDB(b, 100)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := adapter.FindMany(ctx, "test_items", nil, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGormAdapterCreate(b *testing.B) {
	adapter := setupBenchmarkGormDB(b, 0)
	ctx := context.Background()
	namePrefix := strings.ReplaceAll(b.Name(), "/", "_")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if err := adapter.Create(ctx, "test_items", map[string]any{
			"name":  fmt.Sprintf("%s Bench %d", namePrefix, i),
			"email": fmt.Sprintf("%s-bench-%d@example.com", namePrefix, i),
			"age":   i,
		}); err != nil {
			b.Fatal(err)
		}
	}
}
