# Database Schema

Limen stores core auth data and enabled plugin data in your database. The final
schema depends on the plugins and schema options you configure.

## Core Tables

The core library contributes tables for:

- users
- linked accounts
- sessions
- verifications
- database-backed rate limits

Plugins can extend those tables or add their own tables. For example,
credential-password can add `username` to users, two-factor adds two-factor
tables, and session-jwt adds refresh-token tables.

## Generate Schema Metadata

Enable CLI schema export while developing:

```go
auth, err := limen.New(&limen.Config{
	Database: adapter,
	CLI:      &limen.CLIConfig{Enabled: true},
	Plugins: []limen.Plugin{
		credentialpassword.New(),
	},
})
```

Run your app once. Limen writes `.limen/schemas.json`, which the CLI uses for
model and migration generation.

## Generate Migrations

```bash
limen generate migrations \
  --driver postgres \
  --dsn "$DATABASE_URL" \
  --output ./migrations
```

The CLI compares the generated schema metadata with your database and writes SQL
files only when changes are needed.

## Manual Schema

If you do not want to use the CLI, create the same tables manually. The
repository includes a PostgreSQL example schema at
`examples/adapters/sql/auth.sql`.

Keep optional plugin sections aligned with the plugins you actually enable.

## Refresh After Config Changes

Refresh schema metadata and migrations whenever you change:

- enabled plugins
- plugin options that add fields or tables
- schema customization options
- ID generation strategy

After changing configuration, run the app once, regenerate migrations, and apply
them through your normal migration tool.
