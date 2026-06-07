# Gin

Limen exposes an `http.Handler`, so Gin integration is a small adapter route:
forward the auth base path to `auth.Handler()` and use `auth.GetSession(r)` in
your own handlers.

## Install

```bash
go get github.com/gin-gonic/gin
go get github.com/ragokan/limen
go get github.com/ragokan/limen/adapters/gorm
go get github.com/ragokan/limen/plugins/credential-password
```

## Mount Limen

```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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

	r := gin.Default()

	r.GET("/api/profile", func(c *gin.Context) {
		session, err := auth.GetSession(c.Request)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user":    session.User,
			"session": session.Session,
		})
	})

	r.Any("/api/auth/*path", func(c *gin.Context) {
		auth.Handler().ServeHTTP(c.Writer, c.Request)
	})

	log.Fatal(r.Run(":8080"))
}
```

## Route Matching

The Gin wildcard route must match the Limen HTTP base path. If you configure:

```go
limen.WithHTTPBasePath("/api/auth")
```

then forward:

```go
r.Any("/api/auth/*path", ...)
```

If you choose a different base path, update both values together.
