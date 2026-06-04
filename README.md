<p align="center">
  <a href="https://limenauth.dev">
    <img src="./banner.svg" alt="Limen — Composable authentication for Go" width="640" />
  </a>
</p>

<p align="center">
  A modern, composable authentication library for Go, inspired by <a href="https://www.better-auth.com/">better-auth</a>.
</p>

<p align="center">
  <a href="https://limenauth.dev">Documentation</a>
  ·
  <a href="https://github.com/ragokan/limen/issues">Issues</a>
  ·
  <a href="https://x.com/limenauth">X (@limenauth)</a>
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/ragokan/limen"><img src="https://img.shields.io/badge/reference-pkg.go.dev-ffffff?style=flat&colorA=000000&colorB=000000&logo=go&logoColor=white" alt="Go reference" /></a>
  <a href="https://github.com/ragokan/limen/stargazers"><img src="https://img.shields.io/github/stars/thecodearcher/limen?style=flat&colorA=000000&colorB=000000&logo=github" alt="GitHub stars" /></a>

</p>

Limen is a modular authentication library for Go that takes a **plugin-first** approach — the core ships with interfaces, session management, and security primitives, while every authentication method lives in its own importable Go module. You compose exactly the auth stack your application needs without pulling in code/dependencies you don't use.

Out of the box, Limen provides:

- Credential/password authentication
- OAuth 2.0
- Two-factor authentication
- Session management
- Optional cache-backed sessions and rate limiting
- CLI schema export and migration generation
- ...and more

Bring your own database, bring your own framework — Limen adapts to your stack, not the other way around.

## Documentation

Full guides, configuration reference, and plugin documentation are available at **[limenauth.dev](https://limenauth.dev)**.

## Requirements

- Go 1.25+

## Installation

```bash
go get github.com/ragokan/limen
```

Then add the adapter and plugins your application needs:

```bash
go get github.com/ragokan/limen/adapters/gorm
go get github.com/ragokan/limen/plugins/credential-password
```

## Quick Start

```go
package main

import (
	"log"
	"net/http"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/ragokan/limen"
	gormadapter "github.com/ragokan/limen/adapters/gorm"
	credentialpassword "github.com/ragokan/limen/plugins/credential-password"
)

func main() {
	db, err := gorm.Open(postgres.Open("your-dsn"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	auth, err := limen.New(&limen.Config{
		BaseURL:  "http://localhost:8080",
		Database: gormadapter.New(db),
		Secret:   []byte("your-32-byte-secret-key-here!!!!"),
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

Alternatively, set the `LIMEN_SECRET` environment variable and omit the `Secret` from the struct.

For a more complete example with OAuth providers, two-factor auth, and Gin integration, see the [examples](examples/).

For full configuration options, usage, and plugin APIs, visit **[limenauth.dev](https://limenauth.dev)**.

## Development

This repository is a Go workspace. The checked-in [go.work](go.work) file makes
root, adapter, plugin, CLI, and example modules resolve to the current branch.

Run all non-example module tests:

```bash
./scripts/test-modules.sh
```

Run every module outside `go.work`, including examples:

```bash
./scripts/test-standalone-modules.sh --all
```

Run benchmark suites and PostgreSQL 18 integration tests:

```bash
./scripts/run-benchmarks.sh
./scripts/test-postgres18.sh
```

PostgreSQL cleanup:

```go
auth, err := limen.New(&limen.Config{
	Database: db,
	Secret:   secret,
	Cleanup:  limen.NewDefaultCleanupConfig(limen.WithCleanupOnInit(true)),
})
```

```go
err := auth.CleanupExpired(ctx)
```

Provider behavior, verified-email rules, and Instagram support status are
documented in [OAuth Providers](docs/oauth-providers.md). PostgreSQL TTL and
cleanup details are documented in [PostgreSQL Cleanup And TTL](docs/postgres-cleanup.md).
Benchmarking is documented in [Benchmarking](docs/benchmarking.md). Production
deployment guidance is documented in [Production Setup](docs/production.md).
Release/versioning rules are documented in [Releasing](docs/releasing.md).

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## Security

Found a security issue? Please **do not** open a public issue. Email
[security@limenauth.dev](mailto:security@limenauth.dev) instead. See
[SECURITY.md](SECURITY.md) for full details on our disclosure process.

## License

MIT License — see [LICENSE](LICENSE) for details.
