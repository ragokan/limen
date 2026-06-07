# Sessions

Limen creates opaque sessions by default. Session tokens are stored in the
database unless you configure a custom or cache-backed store.

## Defaults

- duration: 7 days
- update age: 1 day
- idle timeout: disabled
- session store: database
- short session duration: 24 hours
- bearer token support: disabled

## Configure Session Lifetime

```go
auth, err := limen.New(&limen.Config{
	Database: adapter,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	Session: limen.NewDefaultSessionConfig(
		limen.WithSessionDuration(14*24*time.Hour),
		limen.WithSessionUpdateAge(24*time.Hour),
		limen.WithSessionIdleTimeout(72*time.Hour),
		limen.WithSessionActivityCheckInterval(15*time.Minute),
	),
})
```

Validation rules:

- update age must be less than or equal to duration
- idle timeout must be less than or equal to duration
- activity check interval must be less than idle timeout when both are enabled
- update age must be less than idle timeout when idle timeout is enabled

## Remember Me

Credential sign-in accepts `remember_me`. When a user signs in with
`remember_me: false`, Limen uses the short session duration.

```go
Session: limen.NewDefaultSessionConfig(
	limen.WithSessionShortDuration(8*time.Hour),
)
```

Set the short duration to `0` to disable short-session behavior.

## Bearer Tokens

Enable bearer-token support for API clients that cannot use cookies:

```go
Session: limen.NewDefaultSessionConfig(
	limen.WithBearerEnabled(),
)
```

When enabled, Limen accepts:

```http
Authorization: Bearer <session-token>
```

Session responses also include token headers alongside cookies.

## Session Metadata

Limen stores request metadata such as IP address and user agent with sessions.
Override extractors when your app runs behind proxies or needs custom metadata:

```go
Session: limen.NewDefaultSessionConfig(
	limen.WithSessionIPAddressExtractor(func(r *http.Request) string {
		return r.Header.Get("X-Real-IP")
	}),
	limen.WithSessionUserAgentExtractor(func(r *http.Request) string {
		return r.UserAgent()
	}),
)
```

For proxy-aware IP extraction, use
`limen.NewTrustedProxyIPExtractor`.
