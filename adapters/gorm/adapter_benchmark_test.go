package gorm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

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

func setupPostgresBenchmarkGormDB(b *testing.B, rows int) (*Adapter, limen.SchemaTableName) {
	b.Helper()
	dsn := os.Getenv("LIMEN_POSTGRES_DSN")
	if dsn == "" {
		b.Skip("set LIMEN_POSTGRES_DSN to run PostgreSQL benchmarks")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		b.Fatalf("open postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		b.Fatalf("db handle: %v", err)
	}
	b.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	table := limen.SchemaTableName(fmt.Sprintf("limen_gorm_bench_%d", time.Now().UnixNano()))
	quoted := `"` + strings.ReplaceAll(string(table), `"`, `""`) + `"`
	b.Cleanup(func() { _ = db.WithContext(ctx).Exec("DROP TABLE IF EXISTS " + quoted).Error })
	if err := db.WithContext(ctx).Exec("DROP TABLE IF EXISTS " + quoted).Error; err != nil {
		b.Fatalf("drop table: %v", err)
	}
	if err := db.WithContext(ctx).Exec(fmt.Sprintf(`CREATE TABLE %s (
		"id" BIGSERIAL PRIMARY KEY,
		"name" TEXT NOT NULL,
		"email" TEXT UNIQUE,
		"age" INTEGER DEFAULT 0
	)`, quoted)).Error; err != nil {
		b.Fatalf("create table: %v", err)
	}

	adapter := New(db)
	for i := range rows {
		if err := adapter.Create(ctx, table, map[string]any{
			"name":  fmt.Sprintf("User %d", i),
			"email": fmt.Sprintf("user-%d@example.com", i),
			"age":   i,
		}); err != nil {
			b.Fatalf("seed item: %v", err)
		}
	}
	return adapter, table
}

func BenchmarkGormPostgresFindOne(b *testing.B) {
	adapter, table := setupPostgresBenchmarkGormDB(b, 100)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := adapter.FindOne(ctx, table, []limen.Where{
			limen.Eq("email", "user-42@example.com"),
		}, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGormPostgresFindMany(b *testing.B) {
	adapter, table := setupPostgresBenchmarkGormDB(b, 100)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := adapter.FindMany(ctx, table, nil, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGormPostgresCreate(b *testing.B) {
	adapter, table := setupPostgresBenchmarkGormDB(b, 0)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		if err := adapter.Create(ctx, table, map[string]any{
			"name":  fmt.Sprintf("Bench %d", i),
			"email": fmt.Sprintf("bench-%d@example.com", i),
			"age":   i,
		}); err != nil {
			b.Fatal(err)
		}
	}
}
