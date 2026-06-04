package limen

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCleanupTestLimen(t *testing.T, adapter DatabaseAdapter, cleanup *cleanupConfig) *Limen {
	t.Helper()
	l, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: adapter,
		Secret:   testSecret,
		Cleanup:  cleanup,
		HTTP: NewDefaultHTTPConfig(
			WithHTTPRateLimiter(
				WithRateLimiterStore(StoreTypeDatabase),
				WithRateLimiterWindow(time.Minute),
			),
		),
	})
	require.NoError(t, err)
	return l
}

func TestCleanupExpiredDeletesExpiredCoreRows(t *testing.T) {
	t.Parallel()

	l := newCleanupTestLimen(t, newTestMemoryAdapter(t), NewDefaultCleanupConfig(WithCleanupOnInit(false)))
	ctx := context.Background()
	now := time.Now()
	seedCleanupRows(t, l, now)

	require.NoError(t, l.CleanupExpired(ctx))

	assertSingleCoreRow(t, l, l.core.Schema.Session)
	assertSingleCoreRow(t, l, l.core.Schema.Verification)
	assertSingleCoreRow(t, l, l.core.Schema.RateLimit)
}

func TestCleanupExpiredRunsOnInitByDefault(t *testing.T) {
	t.Parallel()

	adapter := newTestMemoryAdapter(t)
	now := time.Now()
	seedRawExpiredSession(t, adapter, now)

	l := newCleanupTestLimen(t, adapter, nil)

	require.Eventually(t, func() bool {
		count, err := l.core.Count(context.Background(), l.core.Schema.Session, nil)
		return err == nil && count == 0
	}, time.Second, 10*time.Millisecond)
}

func TestCleanupExpiredCanDisableInitCleanup(t *testing.T) {
	t.Parallel()

	adapter := newTestMemoryAdapter(t)
	now := time.Now()
	seedRawExpiredSession(t, adapter, now)

	l := newCleanupTestLimen(t, adapter, NewDefaultCleanupConfig(WithCleanupOnInit(false)))

	assertSingleCoreRow(t, l, l.core.Schema.Session)
}

func TestListSessionsReturnsOnlyActiveSessions(t *testing.T) {
	t.Parallel()

	l := newCleanupTestLimen(t, newTestMemoryAdapter(t), NewDefaultCleanupConfig(WithCleanupOnInit(false)))
	ctx := context.Background()
	now := time.Now()
	userID := int64(1)

	require.NoError(t, l.core.Create(ctx, l.core.Schema.Session, &Session{
		Token:      "active",
		UserID:     userID,
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Hour),
		LastAccess: now,
	}, nil))
	require.NoError(t, l.core.Create(ctx, l.core.Schema.Session, &Session{
		Token:      "expired",
		UserID:     userID,
		CreatedAt:  now.Add(-2 * time.Hour),
		ExpiresAt:  now.Add(-time.Hour),
		LastAccess: now.Add(-time.Hour),
	}, nil))

	sessions, err := l.ListSessions(ctx, userID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "active", sessions[0].Token)
}

func seedCleanupRows(t *testing.T, l *Limen, now time.Time) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, l.core.Create(ctx, l.core.Schema.Session, &Session{
		Token:      "expired-session",
		UserID:     int64(1),
		CreatedAt:  now.Add(-2 * time.Hour),
		ExpiresAt:  now.Add(-time.Hour),
		LastAccess: now.Add(-time.Hour),
	}, nil))
	require.NoError(t, l.core.Create(ctx, l.core.Schema.Session, &Session{
		Token:      "active-session",
		UserID:     int64(1),
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Hour),
		LastAccess: now,
	}, nil))
	require.NoError(t, l.core.Create(ctx, l.core.Schema.Verification, &Verification{
		Subject:   "expired-verification",
		Value:     "expired-value",
		ExpiresAt: now.Add(-time.Hour),
		CreatedAt: now.Add(-2 * time.Hour),
		UpdatedAt: now.Add(-2 * time.Hour),
	}, nil))
	require.NoError(t, l.core.Create(ctx, l.core.Schema.Verification, &Verification{
		Subject:   "active-verification",
		Value:     "active-value",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil))
	require.NoError(t, l.core.Create(ctx, l.core.Schema.RateLimit, &RateLimit{
		Key:           "expired-rate-limit",
		Count:         1,
		LastRequestAt: now.Add(-2 * time.Minute).UnixMilli(),
	}, nil))
	require.NoError(t, l.core.Create(ctx, l.core.Schema.RateLimit, &RateLimit{
		Key:           "active-rate-limit",
		Count:         1,
		LastRequestAt: now.UnixMilli(),
	}, nil))
}

func seedRawExpiredSession(t *testing.T, adapter DatabaseAdapter, now time.Time) {
	t.Helper()
	require.NoError(t, adapter.Create(context.Background(), SessionSchemaTableName, map[string]any{
		string(SchemaIDField):                int64(1),
		string(SessionSchemaTokenField):      "expired-session",
		string(SessionSchemaUserIDField):     int64(1),
		string(SessionSchemaCreatedAtField):  now.Add(-2 * time.Hour),
		string(SessionSchemaExpiresAtField):  now.Add(-time.Hour),
		string(SessionSchemaLastAccessField): now.Add(-time.Hour),
	}))
}

func assertSingleCoreRow(t *testing.T, l *Limen, schema Schema) {
	t.Helper()
	count, err := l.core.Count(context.Background(), schema, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}
