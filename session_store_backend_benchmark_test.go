//go:build bench_integration

package limen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	goredis "github.com/redis/go-redis/v9"
)

type benchmarkPostgresAdapter struct {
	db *sql.DB
}

func newBenchmarkSessionStore(tb testing.TB, storeType StoreType) (context.Context, SessionStore) {
	tb.Helper()

	ctx := context.Background()
	pg := newBenchmarkPostgresAdapter(tb, ctx)
	config := &Config{
		BaseURL:  "http://localhost:8080",
		Database: pg,
		Secret:   testSecret,
		Session: NewDefaultSessionConfig(
			WithSessionStoreType(storeType),
			WithSessionUpdateAge(0),
			WithSessionActivityCheckInterval(0),
		),
	}
	if storeType == StoreTypeCache {
		config.CacheStore = newBenchmarkRedisCache(tb, ctx)
	}

	l, err := New(config)
	if err != nil {
		tb.Fatalf("New: %v", err)
	}
	return ctx, determineStore(l.config.Session, l.core)
}

func newBenchmarkPostgresAdapter(tb testing.TB, ctx context.Context) *benchmarkPostgresAdapter {
	tb.Helper()

	dsn := os.Getenv("LIMEN_POSTGRES_DSN")
	if dsn == "" {
		tb.Skip("set LIMEN_POSTGRES_DSN to run PostgreSQL session store benchmarks")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		tb.Fatalf("open postgres: %v", err)
	}
	db.SetMaxOpenConns(32)
	db.SetMaxIdleConns(32)
	tb.Cleanup(func() { _ = db.Close() })

	if err := db.PingContext(ctx); err != nil {
		tb.Fatalf("ping postgres: %v", err)
	}
	resetBenchmarkSessionSchema(tb, ctx, db)
	return &benchmarkPostgresAdapter{db: db}
}

func newBenchmarkRedisCache(tb testing.TB, ctx context.Context) CacheAdapter {
	tb.Helper()

	addr := os.Getenv("LIMEN_REDIS_ADDR")
	if addr == "" {
		tb.Skip("set LIMEN_REDIS_ADDR to run Redis session store benchmarks")
	}
	client := goredis.NewClient(&goredis.Options{Addr: addr})
	tb.Cleanup(func() { _ = client.Close() })
	if err := client.Ping(ctx).Err(); err != nil {
		tb.Fatalf("ping redis: %v", err)
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		tb.Fatalf("flush redis: %v", err)
	}
	return &benchmarkRedisCache{client: client}
}

func resetBenchmarkSessionSchema(tb testing.TB, ctx context.Context, db *sql.DB) {
	tb.Helper()

	statements := []string{
		`DROP TABLE IF EXISTS sessions`,
		`CREATE TABLE sessions (
			id BIGSERIAL PRIMARY KEY,
			token TEXT NOT NULL UNIQUE,
			user_id BIGINT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			last_access TIMESTAMPTZ NOT NULL,
			metadata TEXT
		)`,
		`CREATE INDEX idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX idx_sessions_expires_at ON sessions(expires_at)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			tb.Fatalf("prepare postgres schema: %v", err)
		}
	}
}

func BenchmarkSessionStoreSet(b *testing.B) {
	b.Run("Postgres", func(b *testing.B) {
		ctx, store := newBenchmarkSessionStore(b, StoreTypeDatabase)
		benchmarkSessionStoreSet(b, ctx, store)
	})
	b.Run("Redis", func(b *testing.B) {
		ctx, store := newBenchmarkSessionStore(b, StoreTypeCache)
		benchmarkSessionStoreSet(b, ctx, store)
	})
}

func BenchmarkSessionStoreGet(b *testing.B) {
	b.Run("Postgres", func(b *testing.B) {
		ctx, store := newBenchmarkSessionStore(b, StoreTypeDatabase)
		benchmarkSessionStoreGet(b, ctx, store)
	})
	b.Run("Redis", func(b *testing.B) {
		ctx, store := newBenchmarkSessionStore(b, StoreTypeCache)
		benchmarkSessionStoreGet(b, ctx, store)
	})
}

func BenchmarkSessionStoreListByUserID(b *testing.B) {
	b.Run("Postgres", func(b *testing.B) {
		ctx, store := newBenchmarkSessionStore(b, StoreTypeDatabase)
		benchmarkSessionStoreListByUserID(b, ctx, store)
	})
	b.Run("Redis", func(b *testing.B) {
		ctx, store := newBenchmarkSessionStore(b, StoreTypeCache)
		benchmarkSessionStoreListByUserID(b, ctx, store)
	})
}

func BenchmarkSessionStoreDeleteByUserID(b *testing.B) {
	b.Run("Postgres", func(b *testing.B) {
		ctx, store := newBenchmarkSessionStore(b, StoreTypeDatabase)
		benchmarkSessionStoreDeleteByUserID(b, ctx, store)
	})
	b.Run("Redis", func(b *testing.B) {
		ctx, store := newBenchmarkSessionStore(b, StoreTypeCache)
		benchmarkSessionStoreDeleteByUserID(b, ctx, store)
	})
}

func benchmarkSessionStoreSet(b *testing.B, ctx context.Context, store SessionStore) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session := benchmarkSession("set-"+strconv.Itoa(i), int64(i+1), now, time.Hour)
		if err := store.Set(ctx, session); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkSessionStoreGet(b *testing.B, ctx context.Context, store SessionStore) {
	now := time.Now()
	session := benchmarkSession("get-token", int64(1), now, time.Hour)
	if err := store.Set(ctx, session); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.Get(ctx, session.Token); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkSessionStoreListByUserID(b *testing.B, ctx context.Context, store SessionStore) {
	now := time.Now()
	userID := int64(1)
	for i := 0; i < 100; i++ {
		if err := store.Set(ctx, benchmarkSession("list-"+strconv.Itoa(i), userID, now, time.Hour)); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sessions, err := store.ListByUserID(ctx, userID)
		if err != nil {
			b.Fatal(err)
		}
		if len(sessions) != 100 {
			b.Fatalf("sessions = %d, want 100", len(sessions))
		}
	}
}

func benchmarkSessionStoreDeleteByUserID(b *testing.B, ctx context.Context, store SessionStore) {
	now := time.Now()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userID := int64(i + 1)
		b.StopTimer()
		for j := 0; j < 10; j++ {
			token := "delete-" + strconv.Itoa(i) + "-" + strconv.Itoa(j)
			if err := store.Set(ctx, benchmarkSession(token, userID, now, time.Hour)); err != nil {
				b.Fatal(err)
			}
		}
		b.StartTimer()
		if err := store.DeleteByUserID(ctx, userID); err != nil {
			b.Fatal(err)
		}
	}
}

func TestSessionStoreRedisTTLExpires(t *testing.T) {
	ctx, store := newBenchmarkSessionStore(t, StoreTypeCache)
	session := benchmarkSession("ttl-token", int64(1), time.Now(), 75*time.Millisecond)
	if err := store.Set(ctx, session); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, err := store.Get(ctx, session.Token); err != nil {
		t.Fatalf("Get before TTL: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, err := store.Get(ctx, session.Token)
		if errors.Is(err, ErrSessionNotFound) {
			return
		}
		if err != nil {
			t.Fatalf("Get after TTL: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("session still exists after TTL")
}

func benchmarkSession(token string, userID int64, now time.Time, ttl time.Duration) *Session {
	return &Session{
		Token:      token,
		UserID:     userID,
		CreatedAt:  now,
		ExpiresAt:  now.Add(ttl),
		LastAccess: now,
		Metadata: map[string]any{
			"bench": true,
		},
	}
}

func (a *benchmarkPostgresAdapter) Create(ctx context.Context, tableName SchemaTableName, data map[string]any) error {
	if tableName != SessionSchemaTableName {
		return fmt.Errorf("unsupported table: %s", tableName)
	}
	_, err := a.db.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, created_at, expires_at, last_access, metadata) VALUES ($1, $2, $3, $4, $5, $6)`,
		data[string(SessionSchemaTokenField)],
		data[string(SessionSchemaUserIDField)],
		data[string(SessionSchemaCreatedAtField)],
		data[string(SessionSchemaExpiresAtField)],
		data[string(SessionSchemaLastAccessField)],
		data[string(SessionSchemaMetadataField)],
	)
	return err
}

func (a *benchmarkPostgresAdapter) FindOne(ctx context.Context, tableName SchemaTableName, conditions []Where, _ []OrderBy) (map[string]any, error) {
	if tableName != SessionSchemaTableName {
		return nil, fmt.Errorf("unsupported table: %s", tableName)
	}
	token, ok := benchmarkEqValue(conditions, string(SessionSchemaTokenField))
	if !ok {
		return nil, fmt.Errorf("unsupported conditions: %v", conditions)
	}
	row := a.db.QueryRowContext(ctx,
		`SELECT id, token, user_id, created_at, expires_at, last_access, metadata FROM sessions WHERE token = $1 LIMIT 1`,
		token,
	)
	return benchmarkScanSession(row)
}

func (a *benchmarkPostgresAdapter) FindMany(ctx context.Context, tableName SchemaTableName, conditions []Where, _ *QueryOptions) ([]map[string]any, error) {
	if tableName != SessionSchemaTableName {
		return nil, fmt.Errorf("unsupported table: %s", tableName)
	}
	userID, ok := benchmarkEqValue(conditions, string(SessionSchemaUserIDField))
	if !ok {
		return nil, fmt.Errorf("unsupported conditions: %v", conditions)
	}
	rows, err := a.db.QueryContext(ctx,
		`SELECT id, token, user_id, created_at, expires_at, last_access, metadata FROM sessions WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0, 100)
	for rows.Next() {
		session, err := benchmarkScanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, session)
	}
	return out, rows.Err()
}

func (a *benchmarkPostgresAdapter) Update(ctx context.Context, tableName SchemaTableName, conditions []Where, updates map[string]any) error {
	if tableName != SessionSchemaTableName {
		return fmt.Errorf("unsupported table: %s", tableName)
	}
	token, ok := benchmarkEqValue(conditions, string(SessionSchemaTokenField))
	if !ok {
		return fmt.Errorf("unsupported conditions: %v", conditions)
	}
	_, err := a.db.ExecContext(ctx,
		`UPDATE sessions SET token = $1, user_id = $2, created_at = $3, expires_at = $4, last_access = $5, metadata = $6 WHERE token = $7`,
		updates[string(SessionSchemaTokenField)],
		updates[string(SessionSchemaUserIDField)],
		updates[string(SessionSchemaCreatedAtField)],
		updates[string(SessionSchemaExpiresAtField)],
		updates[string(SessionSchemaLastAccessField)],
		updates[string(SessionSchemaMetadataField)],
		token,
	)
	return err
}

func (a *benchmarkPostgresAdapter) Delete(ctx context.Context, tableName SchemaTableName, conditions []Where) error {
	if tableName != SessionSchemaTableName {
		return fmt.Errorf("unsupported table: %s", tableName)
	}
	if token, ok := benchmarkEqValue(conditions, string(SessionSchemaTokenField)); ok {
		_, err := a.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = $1`, token)
		return err
	}
	if userID, ok := benchmarkEqValue(conditions, string(SessionSchemaUserIDField)); ok {
		_, err := a.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID)
		return err
	}
	return fmt.Errorf("unsupported conditions: %v", conditions)
}

func (a *benchmarkPostgresAdapter) Exists(ctx context.Context, tableName SchemaTableName, conditions []Where) (bool, error) {
	if tableName != SessionSchemaTableName {
		return false, fmt.Errorf("unsupported table: %s", tableName)
	}
	n, err := a.Count(ctx, tableName, conditions)
	return n > 0, err
}

func (a *benchmarkPostgresAdapter) Count(ctx context.Context, tableName SchemaTableName, conditions []Where) (int64, error) {
	if tableName != SessionSchemaTableName {
		return 0, fmt.Errorf("unsupported table: %s", tableName)
	}
	if token, ok := benchmarkEqValue(conditions, string(SessionSchemaTokenField)); ok {
		return benchmarkCount(ctx, a.db, `SELECT COUNT(*) FROM sessions WHERE token = $1`, token)
	}
	if userID, ok := benchmarkEqValue(conditions, string(SessionSchemaUserIDField)); ok {
		return benchmarkCount(ctx, a.db, `SELECT COUNT(*) FROM sessions WHERE user_id = $1`, userID)
	}
	return benchmarkCount(ctx, a.db, `SELECT COUNT(*) FROM sessions`)
}

type sessionScanner interface {
	Scan(dest ...any) error
}

func benchmarkScanSession(scanner sessionScanner) (map[string]any, error) {
	var (
		id         int64
		token      string
		userID     int64
		createdAt  time.Time
		expiresAt  time.Time
		lastAccess time.Time
		metadata   sql.NullString
	)
	err := scanner.Scan(&id, &token, &userID, &createdAt, &expiresAt, &lastAccess, &metadata)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	row := map[string]any{
		string(SchemaIDField):                id,
		string(SessionSchemaTokenField):      token,
		string(SessionSchemaUserIDField):     userID,
		string(SessionSchemaCreatedAtField):  createdAt,
		string(SessionSchemaExpiresAtField):  expiresAt,
		string(SessionSchemaLastAccessField): lastAccess,
		string(SessionSchemaMetadataField):   nil,
	}
	if metadata.Valid {
		row[string(SessionSchemaMetadataField)] = metadata.String
	}
	return row, nil
}

func benchmarkEqValue(conditions []Where, column string) (any, bool) {
	for _, condition := range conditions {
		if condition.Column == column && (condition.Operator == "" || condition.Operator == OpEq) {
			return condition.Value, true
		}
	}
	return nil, false
}

func benchmarkCount(ctx context.Context, db *sql.DB, query string, args ...any) (int64, error) {
	var n int64
	err := db.QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}

type benchmarkRedisCache struct {
	client *goredis.Client
}

func (c *benchmarkRedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, ErrRecordNotFound
	}
	return data, err
}

func (c *benchmarkRedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl < 0 {
		ttl = 0
	}
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *benchmarkRedisCache) Has(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	return n > 0, err
}

func (c *benchmarkRedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}
