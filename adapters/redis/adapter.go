// Package redis provides a Redis-backed CacheAdapter for Limen.
package redis

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/ragokan/limen"
)

type Adapter struct {
	client goredis.Cmdable
	close  func() error
}

var _ limen.CacheAdapter = (*Adapter)(nil)

func New(client goredis.Cmdable) *Adapter {
	return &Adapter{client: client}
}

func NewClient(opts *goredis.Options) *Adapter {
	client := goredis.NewClient(opts)
	return &Adapter{
		client: client,
		close:  client.Close,
	}
}

func (a *Adapter) Client() goredis.Cmdable {
	return a.client
}

func (a *Adapter) Close() error {
	if a.close == nil {
		return nil
	}
	return a.close()
}

func (a *Adapter) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := a.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return nil, limen.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (a *Adapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl < 0 {
		ttl = 0
	}
	return a.client.Set(ctx, key, value, ttl).Err()
}

func (a *Adapter) Has(ctx context.Context, key string) (bool, error) {
	n, err := a.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (a *Adapter) Delete(ctx context.Context, key string) error {
	return a.client.Del(ctx, key).Err()
}
