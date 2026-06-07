# SQL Adapter

Use the SQL adapter when your application already owns a `*sql.DB` connection.
The adapter wraps that connection so Limen can store users, sessions,
verifications, rate-limit state, and plugin data in the same database.

The adapter uses `sqlx` internally while keeping your application on standard
`database/sql`.

## Install

```bash
go get github.com/ragokan/limen/adapters/sql
go get github.com/lib/pq
```

Use the driver package for your database:

- PostgreSQL: `github.com/lib/pq`
- MySQL: `github.com/go-sql-driver/mysql`
- SQLite: `modernc.org/sqlite`

## Create The Adapter

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
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	auth, err := limen.New(&limen.Config{
		BaseURL:  "http://localhost:8080",
		Database: sqladapter.NewPostgreSQL(db),
		Secret:   []byte(os.Getenv("LIMEN_SECRET")),
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
	log.Fatal(http.ListenAndServe(":8080", mux))
}
```

`LIMEN_SECRET` must be exactly 32 bytes. For local development:

```bash
export LIMEN_SECRET="$(openssl rand -hex 16)"
```

Choose the constructor that matches your database:

```go
sqladapter.NewPostgreSQL(db)
sqladapter.NewMySQL(db)
sqladapter.NewSQLite(db)
```

## Migrations

Limen does not create database tables automatically. Generate SQL migrations
with the [CLI](../concepts/cli.md), then apply them with your migration tool.

```bash
limen generate migrations \
  --driver postgres \
  --dsn "$DATABASE_URL" \
  --output ./migrations
```

The CLI currently generates migrations for PostgreSQL and MySQL.

## Query Logging

For profiling or debugging, attach a query logger:

```go
adapter := sqladapter.NewPostgreSQL(db).WithLogger(myLogger)
```

The logger must implement:

```go
type QueryLogger interface {
	LogQuery(ctx context.Context, query string, args any, duration time.Duration, err error)
}
```

## MySQL Notes

The MySQL driver's text protocol can return `[]byte` for string columns. The
SQL adapter normalizes those values to `string` after scanning so text and
prepared protocols behave consistently.
