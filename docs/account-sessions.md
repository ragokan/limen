# Accounts And Sessions

Limen exposes session management both as Go APIs and HTTP routes.

## Session APIs

```go
session, err := auth.GetSession(req)
sessions, err := auth.ListSessions(ctx, userID)
err := auth.RevokeSession(ctx, token)
err := auth.RevokeAllSessions(ctx, userID)
```

HTTP routes under the Limen base path:

```text
GET  /auth/me
GET  /auth/sessions
POST /auth/signout
POST /auth/revoke-sessions
```

`GET /auth/sessions` returns a redacted session list for the current user. It
includes session IDs, user IDs, timestamps, and request metadata such as
`ip_address` and `user_agent` when available. It does not include session
tokens, refresh tokens, access tokens, or arbitrary session metadata.

`/auth/revoke-sessions` revokes all sessions for the current user.

## OAuth Linked Accounts

When the OAuth plugin is enabled:

```text
GET    /auth/oauth/accounts
GET    /auth/oauth/:provider/link
DELETE /auth/oauth/:provider/unlink
GET    /auth/oauth/:provider/tokens
POST   /auth/oauth/:provider/tokens/refresh
```

Access tokens, refresh tokens, and ID tokens are encrypted at rest in the account
table. Token endpoints require the current user session.

## Bearer Sessions

Opaque sessions use cookies by default. Enable bearer support only for clients
that cannot use cookies:

```go
Session: limen.NewDefaultSessionConfig(
	limen.WithBearerEnabled(),
)
```

When bearer support is enabled, session responses can include `Set-Auth-Token`
headers and protected routes accept `Authorization: Bearer <token>`.

API keys use `X-Limen-API-Key` or `Authorization: ApiKey ...`; they do not use
Bearer by default.
