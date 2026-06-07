# OpenAPI

Limen can generate an OpenAPI 3.1 document for all registered auth routes,
including plugin routes and disabled path settings.

## Serve OpenAPI

Configure an OpenAPI route under the Limen HTTP config:

```go
auth, err := limen.New(&limen.Config{
	BaseURL:  "https://api.example.com",
	Database: adapter,
	Secret:   []byte(os.Getenv("LIMEN_SECRET")),
	HTTP: limen.NewDefaultHTTPConfig(
		limen.WithHTTPBasePath("/api/auth"),
		limen.WithHTTPOpenAPI("/openapi.json",
			limen.WithOpenAPITitle("Example Auth API"),
			limen.WithOpenAPIVersion("1.0.0"),
			limen.WithOpenAPIDescription("Authentication endpoints"),
		),
	),
})
```

The document is served at:

```text
GET /api/auth/openapi.json
```

## Generate In Go

```go
doc := auth.OpenAPI(
	limen.WithOpenAPITitle("Example Auth API"),
	limen.WithOpenAPIVersion("1.0.0"),
)

data, err := auth.OpenAPIJSON()
```

The default security schemes include the configured session cookie. If bearer
sessions are enabled, the document also includes bearer auth.

## Servers And Security Schemes

```go
doc := auth.OpenAPI(
	limen.WithOpenAPIServers(limen.OpenAPIServer{
		URL:         "https://api.example.com",
		Description: "Production",
	}),
	limen.WithOpenAPISecurityScheme("apiKey", limen.OpenAPISecurityScheme{
		Type: "apiKey",
		In:   "header",
		Name: "X-API-Key",
	}),
)
```

## Huma Integration

The Huma bridge merges Limen auth routes into an existing Huma OpenAPI
document:

```go
err := limenhuma.Merge(api, auth,
	limen.WithOpenAPITitle("Example API"),
)
```

The bridge detects conflicting path/method operations before merging.

See [OpenAPI And Huma](../../openapi.md) for more detail.
