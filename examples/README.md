# Examples

Small, focused examples demonstrating Limen features. Each example is a standalone Go module that can be copied out of the repo and run independently.

## Prerequisites

- Go 1.25+
- A running PostgreSQL database
- `DATABASE_URL` environment variable set (e.g. `postgres://user:pass@localhost:5432/limen?sslmode=disable`)

## Run

From the repository root (uses `go.work` for local module resolution):

```bash
DATABASE_URL="postgres://..." go run ./examples/basic
DATABASE_URL="postgres://..." go run ./examples/gin
DATABASE_URL="postgres://..." GOOGLE_CLIENT_ID=... GOOGLE_CLIENT_SECRET=... go run ./examples/oauth-google
DATABASE_URL="postgres://..." go run ./examples/two-factor
DATABASE_URL="postgres://..." go run ./examples/adapters/gorm
DATABASE_URL="postgres://..." go run ./examples/adapters/sql
```

## Examples

| Example | Adapter | Plugins | Extra env vars |
|---------|---------|---------|----------------|
| `basic` | `database/sql` | credential-password | -- |
| `gin` | GORM | credential-password | -- |
| `oauth-google` | GORM | OAuth (Google) | `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` |
| `two-factor` | GORM | credential-password, two-factor | -- |
| `adapters/gorm` | GORM | credential-password | -- |
| `adapters/sql` | `database/sql` | credential-password | -- |

The `basic` and `gin` examples also include a custom `GET /api/profile` endpoint that demonstrates how to use `auth.GetSession(r)` to read the authenticated user outside of Limen's built-in routes.

Examples mount Limen under `/api/auth` and set
`limen.WithHTTPBasePath("/api/auth")`, so built-in routes live at paths such as
`/api/auth/signin`, `/api/auth/me`, and `/api/auth/oauth/google/authorize`.

OAuth provider modules, default scopes, verified-email behavior, and unsupported
Instagram sign-in status are summarized in [OAuth Providers](../docs/oauth-providers.md).
Production deployment defaults and security settings are summarized in
[Production Setup](../docs/production.md).

## PostgreSQL 18

For local compatibility checks, run the repository helper from the repository
root:

```bash
./scripts/test-postgres18.sh
```
