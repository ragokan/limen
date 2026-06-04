package limen

import (
	"context"
	"fmt"
	"time"
)

type cleanupConfig struct {
	OnInit bool
}

type CleanupConfigOption func(*cleanupConfig)

func NewDefaultCleanupConfig(opts ...CleanupConfigOption) *cleanupConfig {
	config := &cleanupConfig{
		OnInit: true,
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

func WithCleanupOnInit(enabled bool) CleanupConfigOption {
	return func(c *cleanupConfig) {
		c.OnInit = enabled
	}
}

func (a *Limen) CleanupExpired(ctx context.Context) error {
	return a.core.CleanupExpired(ctx)
}

func (core *LimenCore) CleanupExpired(ctx context.Context) error {
	now := time.Now()
	if err := core.cleanupExpiredSessions(ctx, now); err != nil {
		return err
	}
	if err := core.cleanupExpiredVerifications(ctx, now); err != nil {
		return err
	}
	if err := core.cleanupExpiredRateLimits(ctx, now); err != nil {
		return err
	}
	return nil
}

func (core *LimenCore) cleanupExpiredSessions(ctx context.Context, now time.Time) error {
	schema := core.Schema.Session
	return core.hardDelete(ctx, schema, []Where{
		Lt(schema.GetExpiresAtField(), now),
	})
}

func (core *LimenCore) cleanupExpiredVerifications(ctx context.Context, now time.Time) error {
	schema := core.Schema.Verification
	return core.hardDelete(ctx, schema, []Where{
		Lt(schema.GetExpiresAtField(), now),
	})
}

func (core *LimenCore) cleanupExpiredRateLimits(ctx context.Context, now time.Time) error {
	config := core.config.HTTP.rateLimiter
	if config == nil || config.Store != StoreTypeDatabase {
		return nil
	}
	window, ok := core.maxStaticRateLimitWindow(config)
	if !ok {
		return nil
	}
	schema := core.Schema.RateLimit
	return core.hardDelete(ctx, schema, []Where{
		Lt(schema.GetLastRequestAtField(), now.Add(-window).UnixMilli()),
	})
}

func (core *LimenCore) maxStaticRateLimitWindow(config *RateLimiterConfig) (time.Duration, bool) {
	maxWindow := config.Window
	for _, rule := range config.customRules {
		if rule.limitProvider != nil {
			return 0, false
		}
		if rule.window > maxWindow {
			maxWindow = rule.window
		}
	}
	for _, plugin := range core.config.Plugins {
		for _, rule := range plugin.PluginHTTPConfig().RateLimitRules {
			if rule.limitProvider != nil {
				return 0, false
			}
			if rule.window > maxWindow {
				maxWindow = rule.window
			}
		}
	}
	return maxWindow, true
}

func (core *LimenCore) hardDelete(ctx context.Context, schema Schema, conditions []Where) error {
	if len(conditions) == 0 {
		return fmt.Errorf("%w: conditions required to prevent accidental table-wide delete", ErrMissingConditions)
	}
	return core.getDB(ctx).Delete(ctx, schema.GetTableName(), conditions)
}
