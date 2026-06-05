package sql

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/ragokan/limen"
)

func TestPostgres18AdapterIntegration(t *testing.T) {
	dsn := os.Getenv("LIMEN_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set LIMEN_POSTGRES_DSN to run PostgreSQL integration tests")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	table := limen.SchemaTableName("limen_sql_pg18_" + safeTestName(t.Name()))
	quoted := `"` + strings.ReplaceAll(string(table), `"`, `""`) + `"`
	t.Cleanup(func() { _, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS "+quoted) })
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS "+quoted); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s (
		"id" BIGSERIAL PRIMARY KEY,
		"name" TEXT NOT NULL,
		"email" TEXT,
		"age" INTEGER DEFAULT 0
	)`, quoted)); err != nil {
		t.Fatalf("create table: %v", err)
	}

	adapter := NewPostgreSQL(db)
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
		{Column: "name", Operator: limen.OpContains, Value: 123},
	}, nil); err == nil {
		t.Fatal("expected invalid condition error")
	}
}

func TestPostgres18CleanupExpiredIntegration(t *testing.T) {
	dsn := os.Getenv("LIMEN_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set LIMEN_POSTGRES_DSN to run PostgreSQL integration tests")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	resetCleanupTables(t, ctx, db)

	now := time.Now()
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (token, user_id, created_at, expires_at, last_access) VALUES
		('expired-session', 1, $1, $2, $2),
		('active-session', 1, $3, $4, $3)`,
		now.Add(-2*time.Hour), now.Add(-time.Hour), now, now.Add(time.Hour)); err != nil {
		t.Fatalf("seed sessions: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO verifications (subject, value, expires_at, created_at, updated_at) VALUES
		('expired-verification', 'expired-value', $1, $2, $2),
		('active-verification', 'active-value', $3, $3, $3)`,
		now.Add(-time.Hour), now.Add(-2*time.Hour), now.Add(time.Hour)); err != nil {
		t.Fatalf("seed verifications: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO rate_limits (key, count, last_request_at) VALUES
		('expired-rate-limit', 1, $1),
		('active-rate-limit', 1, $2)`,
		now.Add(-2*time.Minute).UnixMilli(), now.UnixMilli()); err != nil {
		t.Fatalf("seed rate limits: %v", err)
	}

	auth, err := limen.New(&limen.Config{
		BaseURL:  "http://localhost:8080",
		Database: NewPostgreSQL(db),
		Secret:   []byte("01234567890123456789012345678901"),
		Cleanup:  limen.NewDefaultCleanupConfig(limen.WithCleanupOnInit(false)),
		HTTP: limen.NewDefaultHTTPConfig(
			limen.WithHTTPRateLimiter(
				limen.WithRateLimiterStore(limen.StoreTypeDatabase),
				limen.WithRateLimiterWindow(time.Minute),
			),
		),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := auth.CleanupExpired(ctx); err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}

	assertPostgresCount(t, ctx, db, "sessions", 1)
	assertPostgresCount(t, ctx, db, "verifications", 1)
	assertPostgresCount(t, ctx, db, "rate_limits", 1)
	assertPostgresValue(t, ctx, db, "sessions", "token", "active-session")
	assertPostgresValue(t, ctx, db, "verifications", "subject", "active-verification")
	assertPostgresValue(t, ctx, db, "rate_limits", "key", "active-rate-limit")
}

func resetCleanupTables(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	statements := []string{
		`DROP TABLE IF EXISTS sessions`,
		`DROP TABLE IF EXISTS verifications`,
		`DROP TABLE IF EXISTS rate_limits`,
		`CREATE TABLE sessions (
			id BIGSERIAL PRIMARY KEY,
			token TEXT NOT NULL UNIQUE,
			user_id BIGINT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			last_access TIMESTAMPTZ NOT NULL,
			metadata TEXT
		)`,
		`CREATE TABLE verifications (
			id BIGSERIAL PRIMARY KEY,
			subject TEXT NOT NULL,
			value TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			deleted_at TIMESTAMPTZ
		)`,
		`CREATE TABLE rate_limits (
			id BIGSERIAL PRIMARY KEY,
			key TEXT NOT NULL UNIQUE,
			count INTEGER NOT NULL,
			last_request_at BIGINT NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("reset cleanup tables: %v", err)
		}
	}
}

func assertPostgresCount(t *testing.T, ctx context.Context, db *sql.DB, table string, want int) {
	t.Helper()
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "`+table+`"`).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s count = %d, want %d", table, count, want)
	}
}

func assertPostgresValue(t *testing.T, ctx context.Context, db *sql.DB, table, column, want string) {
	t.Helper()
	var got string
	query := fmt.Sprintf("SELECT %q FROM %q LIMIT 1", column, table)
	if err := db.QueryRowContext(ctx, query).Scan(&got); err != nil {
		t.Fatalf("select %s.%s: %v", table, column, err)
	}
	if got != want {
		t.Fatalf("%s.%s = %q, want %q", table, column, got, want)
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
