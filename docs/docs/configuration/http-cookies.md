# HTTP And Cookies

`HTTP` config controls where Limen routes are mounted, cookie behavior, origin
checks, request body limits, response formatting, and OpenAPI serving.

## Base Path

The default auth base path is `/auth`.

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPBasePath("/api/auth"),
)
```

Mount the handler at the same path:

```go
mux.Handle("/api/auth/", auth.Handler())
```

## Trusted Origins

Trusted origins are used for redirect/origin validation and are required when
cross-domain cookies are enabled.

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPTrustedOrigins([]string{
		"https://app.example.com",
	}),
)
```

## Cookie Settings

Defaults:

- cookie name: `limen_session`
- path: `/`
- `Secure`: true
- `HttpOnly`: true
- `SameSite`: Lax
- partitioned: false

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPSessionCookieName("app_session"),
	limen.WithHTTPCookiePath("/"),
	limen.WithHTTPCookieSecure(true),
	limen.WithHTTPCookieHTTPOnly(true),
	limen.WithHTTPCookieSameSite(http.SameSiteLaxMode),
)
```

For local HTTP development, set `Secure` to false only in local builds:

```go
limen.WithHTTPCookieSecure(false)
```

## Cross-Subdomain Cookies

Use cross-subdomain cookies when your app and API live on sibling subdomains:

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPCookieCrossSubdomainEnabled(".example.com"),
)
```

## Cross-Domain Cookies

Cross-domain cookies set `SameSite=None`, `Secure=true`, and partitioned cookie
support. You must configure trusted origins.

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPTrustedOrigins([]string{"https://app.example.com"}),
	limen.WithHTTPCookieCrossDomainEnabled(),
)
```

## Request Body Limit

Default JSON/form body limit is 1 MiB:

```go
HTTP: limen.NewDefaultHTTPConfig(
	limen.WithHTTPMaxBodyBytes(2 << 20),
)
```

## Trusted Proxies

Use a trusted proxy extractor when rate limiting or session metadata should use
the original client IP instead of the reverse proxy IP:

```go
ipExtractor, err := limen.NewTrustedProxyIPExtractor(
	limen.WithTrustedProxyCIDRs("10.0.0.0/8"),
	limen.WithTrustedProxyHeaders(
		limen.TrustedProxyHeaderXForwardedFor,
		limen.TrustedProxyHeaderXRealIP,
	),
	limen.WithTrustedProxyIPv6Prefix(64),
)
if err != nil {
	log.Fatal(err)
}

auth, err := limen.New(&limen.Config{
	Session: limen.NewDefaultSessionConfig(
		limen.WithSessionIPAddressExtractor(ipExtractor),
	),
	HTTP: limen.NewDefaultHTTPConfig(
		limen.WithHTTPRateLimiter(
			limen.WithRateLimiterKeyGenerator(ipExtractor),
		),
	),
})
```

Only trust headers from networks you control.
