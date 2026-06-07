# GORM Adapter

Use the GORM adapter when your application already uses `gorm.io/gorm` and you
want Limen to share the same database connection.

## Install

```bash
go get github.com/ragokan/limen/adapters/gorm
go get gorm.io/gorm
go get gorm.io/driver/postgres
```

Use `gorm.io/driver/mysql` or `gorm.io/driver/sqlite` for MySQL or SQLite.

## Create The Adapter

```go
package main

import (
	"log"
	"net/http"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/ragokan/limen"
	gormadapter "github.com/ragokan/limen/adapters/gorm"
	credentialpassword "github.com/ragokan/limen/plugins/credential-password"
)

func main() {
	db, err := gorm.Open(postgres.Open(os.Getenv("DATABASE_URL")), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	auth, err := limen.New(&limen.Config{
		BaseURL:  "http://localhost:8080",
		Database: gormadapter.New(db),
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

## Migrations

Use the [CLI](../concepts/cli.md) to generate Limen migrations and apply them
with your database migration tool. Limen does not rely on GORM `AutoMigrate` for
its own tables.

```bash
limen generate migrations \
  --driver postgres \
  --dsn "$DATABASE_URL" \
  --output ./migrations
```

## When To Use GORM

Choose the GORM adapter when:

- your app already depends on GORM
- you want Limen reads and writes to use the same connection setup
- your application code uses GORM models for its own tables

Choose the [SQL adapter](sql.md) when your app works directly with
`database/sql` or when you want the smallest adapter dependency surface.
