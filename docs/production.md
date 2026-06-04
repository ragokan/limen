# Production Setup

This guide covers the defaults and settings to make explicit before running
Limen behind a public HTTP endpoint.

## Required Settings

Set a public `BaseURL`, a persistent database adapter, and a 32-byte secret.
Prefer `LIMEN_SECRET` or your secret manager over hard-coding the key.

```go
auth, err := limen.New(&limen.Config{
	BaseURL:  "https://auth.example.com",
	Database: db,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
})
```

`BaseURL` is used to build callback URLs and security checks. It should match
the externally visible scheme, host, and port after any load balancer or reverse
proxy.

## Cookies And Origins

Limen defaults to secure, HttpOnly, SameSite=Lax session cookies, CSRF
protection, and origin checks. Keep those defaults for same-site applications.

For a separate frontend origin, add trusted origins:

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPTrustedOrigins([]string{
		"https://app.example.com",
	}),
)
```

For cross-site cookie delivery, enable cross-domain cookies and keep the trusted
origin list tight:

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPTrustedOrigins([]string{"https://app.example.com"}),
	limen.WithHTTPCookieCrossDomainEnabled(),
)
```

Do not disable CSRF or origin checks for browser-facing routes unless another
verified layer enforces the same boundary.

## Sessions

Database-backed sessions are the default. They give durable session state,
central revocation, and simple horizontal scaling through the database.

```go
Session: limen.NewDefaultSessionConfig(
	limen.WithSessionDuration(7 * 24 * time.Hour),
	limen.WithSessionUpdateAge(24 * time.Hour),
	limen.WithSessionIdleTimeout(0),
)
```

If your application is an API that cannot use cookies, explicitly enable bearer
support:

```go
Session: limen.NewDefaultSessionConfig(
	limen.WithBearerEnabled(),
)
```

## Cleanup And TTL

PostgreSQL does not delete expired rows automatically. Limen runs
`CleanupExpired(ctx)` once during initialization by default and also exposes the
same method for your scheduler.

```go
Cleanup: limen.NewDefaultCleanupConfig(
	limen.WithCleanupOnInit(true),
)
```

For multi-instance deployments, keep init cleanup enabled for simple setups or
disable it and schedule cleanup from one process:

```go
Cleanup: limen.NewDefaultCleanupConfig(
	limen.WithCleanupOnInit(false),
)
```

```go
if err := auth.CleanupExpired(ctx); err != nil {
	return err
}
```

See [PostgreSQL Cleanup And TTL](postgres-cleanup.md) for the exact tables and
indexes.

## Rate Limiting

The HTTP rate limiter is enabled by default and uses the shared cache adapter.
The built-in memory cache is process-local. For multi-instance deployments,
choose a shared custom cache or database-backed rate limits.

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPRateLimiter(
		limen.WithRateLimiterMaxRequests(100),
		limen.WithRateLimiterWindow(time.Minute),
		limen.WithRateLimiterKeyGenerator(func(r *http.Request) string {
			return r.RemoteAddr
		}),
	),
)
```

If the app is behind a trusted proxy, use a key generator that reads the proxy's
canonical client IP header. Only trust forwarded headers that your own proxy
sets.

## OAuth

Use provider modules only for providers you enable. For Google, Apple, and
LinkedIn, keep the OIDC scopes enabled so Limen can validate ID tokens and read
trusted email-verification claims. Facebook profile emails are not treated as
verified. Instagram is not bundled as a first-party sign-in provider.

See [OAuth Providers](oauth-providers.md) for provider-specific behavior.

## Deployment Checklist

- Set `BaseURL` to the public HTTPS origin.
- Set a 32-byte `LIMEN_SECRET`.
- Run migrations before deploying code that expects new indexes or columns.
- Keep CSRF and origin checks enabled for browser flows.
- Configure `TrustedOrigins` for separate frontend origins.
- Use a shared rate-limit store or database-backed rate limits for multiple app
  instances.
- Run `CleanupExpired(ctx)` from one scheduled process when init cleanup is
  disabled.
- Register exact OAuth callback URLs with each provider.
