package redis

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/ragokan/limen"
)

func newIntegrationAdapter(t testing.TB) *Adapter {
	t.Helper()
	addr := os.Getenv("LIMEN_REDIS_ADDR")
	if addr == "" {
		t.Skip("set LIMEN_REDIS_ADDR to run Redis integration tests")
	}
	client := goredis.NewClient(&goredis.Options{Addr: addr})
	t.Cleanup(func() { _ = client.Close() })
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("FlushDB: %v", err)
	}
	return New(client)
}

func TestAdapterSetGetHasDelete(t *testing.T) {
	t.Parallel()

	adapter := newIntegrationAdapter(t)
	ctx := context.Background()

	if err := adapter.Set(ctx, "key", []byte("value"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	has, err := adapter.Has(ctx, "key")
	if err != nil {
		t.Fatalf("Has: %v", err)
	}
	if !has {
		t.Fatal("expected key to exist")
	}
	got, err := adapter.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "value" {
		t.Fatalf("Get = %q, want value", got)
	}
	if err := adapter.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = adapter.Get(ctx, "key")
	if !errors.Is(err, limen.ErrRecordNotFound) {
		t.Fatalf("Get missing error = %v, want ErrRecordNotFound", err)
	}
}

func TestAdapterTTL(t *testing.T) {
	t.Parallel()

	adapter := newIntegrationAdapter(t)
	ctx := context.Background()

	if err := adapter.Set(ctx, "ttl-key", []byte("value"), 25*time.Millisecond); err != nil {
		t.Fatalf("Set: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		has, err := adapter.Has(ctx, "ttl-key")
		if err != nil {
			t.Fatalf("Has: %v", err)
		}
		if !has {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("key did not expire")
}

func BenchmarkRedisCacheSetGet(b *testing.B) {
	adapter := newIntegrationAdapter(b)
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; b.Loop(); i++ {
		key := "bench:set-get"
		if err := adapter.Set(ctx, key, []byte("value"), time.Minute); err != nil {
			b.Fatal(err)
		}
		if _, err := adapter.Get(ctx, key); err != nil {
			b.Fatal(err)
		}
	}
}
