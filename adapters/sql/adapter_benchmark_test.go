package sql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ragokan/limen"

	_ "github.com/jackc/pgx/v5/stdlib"
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

func setupPostgresBenchmarkDB(b *testing.B, rows int) (*Adapter, limen.SchemaTableName) {
	b.Helper()
	dsn := os.Getenv("LIMEN_POSTGRES_DSN")
	if dsn == "" {
		b.Skip("set LIMEN_POSTGRES_DSN to run PostgreSQL benchmarks")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		b.Fatalf("open postgres: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	table := limen.SchemaTableName(fmt.Sprintf("limen_sql_bench_%d", time.Now().UnixNano()))
	quoted := `"` + strings.ReplaceAll(string(table), `"`, `""`) + `"`
	b.Cleanup(func() { _, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS "+quoted) })
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS "+quoted); err != nil {
		b.Fatalf("drop table: %v", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s (
		"id" BIGSERIAL PRIMARY KEY,
		"name" TEXT NOT NULL,
		"email" TEXT UNIQUE,
		"age" INTEGER DEFAULT 0
	)`, quoted)); err != nil {
		b.Fatalf("create table: %v", err)
	}
	adapter := NewPostgreSQL(db)
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

func BenchmarkSQLPostgresFindOne(b *testing.B) {
	adapter, table := setupPostgresBenchmarkDB(b, 100)
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

func BenchmarkSQLPostgresFindMany(b *testing.B) {
	adapter, table := setupPostgresBenchmarkDB(b, 100)
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := adapter.FindMany(ctx, table, nil, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLPostgresCreate(b *testing.B) {
	adapter, table := setupPostgresBenchmarkDB(b, 0)
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
