# Session JWT

The session-jwt plugin replaces Limen's default opaque session manager with
JWT-based sessions.

Use it when clients need self-contained access tokens or bearer-token style API
access. Use the default opaque sessions when server-side revocation and minimal
token exposure are more important.

## Install

```bash
go get github.com/ragokan/limen/plugins/session-jwt
```

## Enable

```go
import sessionjwt "github.com/ragokan/limen/plugins/session-jwt"

auth, err := limen.New(&limen.Config{
	BaseURL:  "https://api.example.com",
	Database: adapter,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	Plugins: []limen.Plugin{
		credentialpassword.New(),
		sessionjwt.New(
			sessionjwt.WithAccessTokenDuration(15*time.Minute),
			sessionjwt.WithRefreshTokenDuration(7*24*time.Hour),
		),
	},
})
```

Defaults:

- signing method: HS256
- access token duration: 15 minutes
- refresh token duration: 7 days
- refresh token rotation: enabled
- refresh tokens: enabled
- blacklist: disabled

For HS256, the plugin uses `Config.Secret` when no signing key is configured.

## Refresh Route

When refresh tokens are enabled:

```text
POST /api/auth/refresh
```

The plugin issues new access tokens and, when rotation is enabled, rotates the
refresh token.

## Issuer And Audience

`BaseURL` is used as the default issuer and audience. Configure them explicitly
for multi-domain deployments:

```go
sessionjwt.New(
	sessionjwt.WithIssuer("https://auth.example.com"),
	sessionjwt.WithAudience([]string{"https://api.example.com"}),
)
```

## Claims

Add custom claims:

```go
sessionjwt.New(
	sessionjwt.WithCustomClaims(func(user *limen.User) map[string]any {
		return map[string]any{"tenant": user.Raw()["tenant_id"]}
	}),
)
```

Use `WithSubject` and `WithSubjectResolver` if you do not want the raw user ID
as the JWT subject.

## Revocation

Enable the blacklist to reject revoked JWTs before natural expiration:

```go
sessionjwt.New(
	sessionjwt.WithBlacklistEnabled(true),
	sessionjwt.WithBlacklistStoreType(limen.StoreTypeCache),
)
```

Use `StoreTypeDatabase` when blacklist state must survive cache loss.
