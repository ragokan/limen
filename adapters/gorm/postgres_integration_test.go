package gorm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ragokan/limen"
)

func TestPostgres18AdapterIntegration(t *testing.T) {
	dsn := os.Getenv("LIMEN_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set LIMEN_POSTGRES_DSN to run PostgreSQL integration tests")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	table := limen.SchemaTableName("limen_gorm_pg18_" + safeTestName(t.Name()))
	quoted := `"` + strings.ReplaceAll(string(table), `"`, `""`) + `"`
	t.Cleanup(func() { _ = db.WithContext(ctx).Exec("DROP TABLE IF EXISTS " + quoted).Error })
	if err := db.WithContext(ctx).Exec("DROP TABLE IF EXISTS " + quoted).Error; err != nil {
		t.Fatalf("drop table: %v", err)
	}
	if err := db.WithContext(ctx).Exec(fmt.Sprintf(`CREATE TABLE %s (
		"id" BIGSERIAL PRIMARY KEY,
		"name" TEXT NOT NULL,
		"email" TEXT,
		"age" INTEGER DEFAULT 0
	)`, quoted)).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	adapter := New(db)
	if err := adapter.Create(ctx, table, map[string]any{"name": "Alice", "email": "alice@example.com", "age": 30}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	found, err := adapter.FindOne(ctx, table, []limen.Where{limen.Eq("name", "Alice")}, nil)
	if err != nil {
		t.Fatalf("FindOne: %v", err)
	}
	if found["email"] != "alice@example.com" {
		t.Fatalf("email = %#v", found["email"])
	}
	exists, err := adapter.Exists(ctx, table, []limen.Where{limen.Eq("name", "Alice")})
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected row to exist")
	}
	if _, err := adapter.FindMany(ctx, table, []limen.Where{
		limen.Eq("name;DROP TABLE test_items", "Alice"),
	}, nil); err == nil {
		t.Fatal("expected invalid condition error")
	}
}

func safeTestName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteRune('_')
	}
	return b.String()
}
