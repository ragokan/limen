package limen

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCacheSessionStore() *cacheSessionStore {
	return &cacheSessionStore{
		cache:  NewMemoryCacheStore(),
		prefix: "test",
	}
}

func TestCacheSessionStoreConcurrentSetPreservesUserIndex(t *testing.T) {
	t.Parallel()

	store := newTestCacheSessionStore()
	ctx := context.Background()
	expiresAt := time.Now().Add(time.Hour)

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = store.Set(ctx, &Session{
				Token:     fmt.Sprintf("token-%d", i),
				UserID:    "user-1",
				CreatedAt: time.Now(),
				ExpiresAt: expiresAt,
			})
		}(i)
	}
	wg.Wait()

	sessions, err := store.ListByUserID(ctx, "user-1")
	require.NoError(t, err)
	assert.Len(t, sessions, 50)
}

func TestCacheSessionStoreListPrunesExpiredAndMissingSessions(t *testing.T) {
	t.Parallel()

	store := newTestCacheSessionStore()
	ctx := context.Background()
	live := Session{
		Token:     "live",
		UserID:    "user-1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	expired := Session{
		Token:     "expired",
		UserID:    "user-1",
		CreatedAt: time.Now().Add(-2 * time.Hour),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	missing := Session{
		Token:     "missing",
		UserID:    "user-1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}

	require.NoError(t, store.Set(ctx, &live))
	require.NoError(t, store.saveUserIndex(ctx, "user-1", []Session{live, expired, missing}))

	sessions, err := store.ListByUserID(ctx, "user-1")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "live", sessions[0].Token)
}

type failingDeleteCacheAdapter struct {
	CacheAdapter
	failKey string
	err     error
}

func (f *failingDeleteCacheAdapter) Delete(ctx context.Context, key string) error {
	if key == f.failKey {
		return f.err
	}
	return f.CacheAdapter.Delete(ctx, key)
}

func TestCacheSessionStoreDeleteByUserIDReturnsTokenDeleteError(t *testing.T) {
	t.Parallel()

	base := NewMemoryCacheStore()
	store := &cacheSessionStore{
		cache:  base,
		prefix: "test",
	}
	ctx := context.Background()
	session := &Session{
		Token:     "token-1",
		UserID:    "user-1",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, store.Set(ctx, session))

	deleteErr := errors.New("delete failed")
	store.cache = &failingDeleteCacheAdapter{
		CacheAdapter: base,
		failKey:      store.sessionKey(session.Token),
		err:          deleteErr,
	}

	err := store.DeleteByUserID(ctx, "user-1")
	require.ErrorIs(t, err, deleteErr)

	hasIndex, err := base.Has(ctx, store.userSessionsKey("user-1"))
	require.NoError(t, err)
	assert.True(t, hasIndex)
}
