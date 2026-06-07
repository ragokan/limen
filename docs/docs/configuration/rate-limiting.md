# Rate Limiting

Limen enables rate limiting by default. The global default is 100 requests per
minute, with plugin-specific rules for sensitive auth routes.

## Configure Defaults

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPRateLimiter(
		limen.WithRateLimiterMaxRequests(60),
		limen.WithRateLimiterWindow(time.Minute),
	),
)
```

Disable the limiter:

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPRateLimiter(
		limen.WithRateLimiterEnabled(false),
	),
)
```

## Stores

The rate limiter uses the cache store by default. Use the database store when
you want rate-limit state persisted in the database:

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPRateLimiter(
		limen.WithRateLimiterStore(limen.StoreTypeDatabase),
	),
)
```

Use `WithRateLimiterCustomStore` for your own implementation.

## Custom Rules

Set a rule for a path or route pattern:

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPRateLimiter(
		limen.WithRateLimiterCustomRule("/signin/credential", 5, 10*time.Second),
	),
)
```

Disable rate limiting for selected paths:

```go
limen.WithRateLimiterDisableForPaths("/healthz")
```

Use a dynamic limit provider when the limit depends on the request:

```go
limen.WithRateLimiterCustomRuleWithLimitProvider(
	"/signin/credential",
	func(r *http.Request) (int, time.Duration) {
		return 5, time.Minute
	},
)
```

## Key Generation

By default, Limen keys rate limits by request IP. Override this when running
behind a trusted proxy or when you need tenant/user-aware keys:

```go
limen.WithRateLimiterKeyGenerator(func(r *http.Request) string {
	return r.Header.Get("X-Forwarded-For")
})
```

For proxy-aware IP extraction, prefer
`limen.NewTrustedProxyIPExtractor` and pass the returned function as the key
generator.
