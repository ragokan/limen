# API Keys

The API-key plugin provides machine-to-machine authentication for CLIs,
background workers, cron jobs, partner integrations, webhook clients, and
service calls.

API keys are not browser sessions. They are long-lived credentials owned by a
user and should be scoped, revocable, and optionally expiring.

## Install

```bash
go get github.com/ragokan/limen/plugins/api-key
```

```go
import apikey "github.com/ragokan/limen/plugins/api-key"

auth, err := limen.New(&limen.Config{
	Database: adapter,
	Secret:   secret,
	Plugins: []limen.Plugin{
		apikey.New(apikey.WithAllowedScopes("jobs:read", "jobs:write")),
	},
})
```

## Management Routes

The plugin mounts under `/api-keys` relative to Limen's HTTP base path.

```text
POST   /auth/api-keys
GET    /auth/api-keys
DELETE /auth/api-keys/:id
```

These routes require a normal user session.

Create request:

```json
{
  "name": "CLI",
  "scopes": ["jobs:read", "jobs:write"],
  "expires_at": "2026-12-31T00:00:00Z"
}
```

The plaintext key is returned only once in the create response. Limen stores a
short lookup prefix plus an HMAC-SHA256 hash using the Limen secret.

Configure every scope that clients may request with `WithAllowedScopes`.
Unconfigured or syntactically invalid scopes are rejected at creation time.
Scopes may contain letters, numbers, `:`, `.`, `_`, `-`, and `/`.

## Authenticate Requests

Use the plugin middleware on your own service routes:

```go
apiKeys := apikey.Use(auth)

mux.Handle("/jobs", apiKeys.MiddlewareRequireAPIKey("jobs:write")(
	http.HandlerFunc(handleJobs),
))
```

Clients can send either:

```text
X-Limen-API-Key: limen_sk_...
Authorization: ApiKey limen_sk_...
```

`Authorization: Bearer` is reserved for Limen session bearer tokens, so API keys
do not use Bearer by default.

Inside handlers:

```go
key, ok := apikey.GetAPIKeyFromContext(r.Context())
```

## Validation

Validation checks:

- key hash with constant-time comparison
- revocation state
- expiry
- required scopes

Successful validation updates `last_used_at`.
