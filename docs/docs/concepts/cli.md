# CLI

The Limen CLI is a development tool for generating Go model structs and SQL
migration files from your configured Limen schemas.

You do not need the CLI at runtime in production.

## Install

```bash
go install github.com/ragokan/limen/cmd/limen@latest
```

Or build it from this repository:

```bash
cd cmd/limen
go build -o limen
```

## Enable Schema Export

Add CLI support to your Limen config:

```go
auth, err := limen.New(&limen.Config{
	Database: adapter,
	CLI:      &limen.CLIConfig{Enabled: true},
	Plugins: []limen.Plugin{
		credentialpassword.New(),
	},
})
```

Run your app once. Limen writes the discovered schema data to
`.limen/schemas.json`.

## Command Shape

```bash
limen [global-flags] generate <subcommand> [flags]
```

Global flag:

- `--schemas`, `-s`: path to the schema file, defaulting to
  `./.limen/schemas.json`

## Generate Models

Generate Go structs for the tables used by your Limen configuration:

```bash
limen generate models
```

Useful flags:

- `--output`, `-o`: output directory, defaulting to `./models`
- `--package`, `-p`: generated package name

Examples:

```bash
limen generate models -o ./internal/models
limen generate models -o ./internal/models -p authmodels
limen -s ./config/limen-schemas.json generate models
```

## Generate Migrations

Generate SQL migration files by comparing `.limen/schemas.json` with your
database:

```bash
limen generate migrations \
  --driver postgres \
  --dsn "postgres://user:pass@localhost:5432/mydb?sslmode=disable" \
  --output ./migrations
```

Useful flags:

- `--driver`, `-d`: database driver, currently `postgres` or `mysql`
- `--dsn`, `-c`: database connection string
- `--output`, `-o`: output directory, defaulting to `./migrations`

For MySQL:

```bash
limen generate migrations \
  --driver mysql \
  --dsn "user:pass@tcp(localhost:3306)/mydb" \
  --output ./migrations
```

Each schema change is generated as an `.up.sql` and `.down.sql` pair. The CLI
does not apply migrations; use `golang-migrate`, `goose`, or your deployment
tooling.

## Recommended Workflow

1. Configure Limen with the adapters, plugins, and schema options your app uses.
2. Enable `CLIConfig` locally.
3. Run the app once to refresh `.limen/schemas.json`.
4. Generate models if your app wants typed structs.
5. Generate migrations for the target database.
6. Apply migrations with your migration tool.
7. Disable CLI export in production if you do not need schema files there.

## Troubleshooting

If the CLI cannot find schemas, run the app once with `CLIConfig` enabled or
pass `--schemas` to the file you want to use.

If no migrations are generated, the database may already match the schema file.
Refresh `.limen/schemas.json` after changing plugins or schema config.

If migration generation fails with an unknown driver, use `postgres` or `mysql`.
