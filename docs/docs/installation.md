# Installation

Add Limen to a Go application by installing the core package, one database
adapter, and at least one authentication plugin.

## Requirements

- Go 1.25 or later
- A database that your chosen adapter supports
- A 32-byte signing secret

## Add Dependencies

This example uses PostgreSQL, the `database/sql` adapter, and the
credential-password plugin.

```bash
go get github.com/ragokan/limen
go get github.com/ragokan/limen/adapters/sql
go get github.com/ragokan/limen/plugins/credential-password
go get github.com/lib/pq
```

For GORM, install `github.com/ragokan/limen/adapters/gorm`,
`gorm.io/gorm`, and the GORM driver for your database.

## Configure Secrets

Set `LIMEN_SECRET` to exactly 32 bytes. Limen also accepts `Config.Secret`
directly, but environment variables are easier to keep out of source code.

```bash
export LIMEN_SECRET="$(openssl rand -hex 16)"
```

## Create The Auth Server

```go
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/ragokan/limen"
	sqladapter "github.com/ragokan/limen/adapters/sql"
	credentialpassword "github.com/ragokan/limen/plugins/credential-password"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("set DATABASE_URL")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	auth, err := limen.New(&limen.Config{
		BaseURL:  "http://localhost:8080",
		Database: sqladapter.NewPostgreSQL(db),
		CLI:      &limen.CLIConfig{Enabled: true},
		HTTP: limen.NewDefaultHTTPConfig(
			limen.WithHTTPBasePath("/api/auth"),
		),
		Plugins: []limen.Plugin{
			credentialpassword.New(),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/auth/", auth.Handler())

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

`CLIConfig` writes `.limen/schemas.json` when the app starts. Keep `.limen/`
out of version control; this repository already ignores it.

## Run Migrations

Install the CLI:

```bash
go install github.com/ragokan/limen/cmd/limen@latest
```

Start your app once so Limen writes `.limen/schemas.json`, then generate SQL
migrations:

```bash
limen generate migrations \
  --driver postgres \
  --dsn "$DATABASE_URL" \
  --output ./migrations
```

Apply the generated SQL files with your migration tool, such as
`golang-migrate`, `goose`, or your own migration runner.

## Run The App

```bash
DATABASE_URL="postgres://user:pass@localhost:5432/limen?sslmode=disable" \
LIMEN_SECRET="$(openssl rand -hex 16)" \
go run .
```

Limen routes are now mounted under `/api/auth`.

Next: [Basic Usage](basic-usage.md).
