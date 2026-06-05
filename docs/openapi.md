# OpenAPI And Huma

Limen can generate an OpenAPI 3.1 document from its registered auth routes.
Route metadata is registered through `RouteBuilder`, so disabled routes, plugin
base paths, and plugin HTTP overrides are reflected in the generated spec.

## Serve OpenAPI From Limen

Enable the OpenAPI endpoint from HTTP config:

```go
auth, err := limen.New(&limen.Config{
	BaseURL:  "https://api.example.com",
	Database: adapter,
	Secret:   secret,
	HTTP: limen.NewDefaultHTTPConfig(
		limen.WithHTTPBasePath("/api/auth"),
		limen.WithHTTPOpenAPI("/openapi.json",
			limen.WithOpenAPITitle("Example Auth API"),
			limen.WithOpenAPIVersion("1.0.0"),
		),
	),
})
```

This serves:

```text
GET /api/auth/openapi.json
```

The generated document intentionally excludes the OpenAPI endpoint itself.

## Use The OpenAPI Handler Directly

You can also mount the handler yourself:

```go
mux.Handle("/auth/openapi.json", auth.OpenAPIHandler(
	limen.WithOpenAPITitle("Example Auth API"),
	limen.WithOpenAPIVersion("1.0.0"),
))
```

## Huma Integration

Huma generates API docs from Huma operations. Limen keeps its own router for auth
routes, so the integration merges Limen's generated OpenAPI paths/components
into a Huma API spec instead of replacing Limen routing.

Use the optional module:

```bash
go get github.com/ragokan/limen/integrations/huma
```

```go
package main

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	limenhuma "github.com/ragokan/limen/integrations/huma"
)

func main() {
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Example API", "1.0.0"))

	auth := newLimen()
	if err := limenhuma.Merge(api, auth); err != nil {
		panic(err)
	}

	mux.Handle("/api/auth/", http.StripPrefix("/api", auth.Handler()))
	http.ListenAndServe(":3000", mux)
}
```

Huma still serves its configured docs endpoints, such as `/docs` and
`/openapi.json`, while Limen handles the auth runtime endpoints.

## Route Metadata

Plugins can enrich generated docs without changing handler signatures:

```go
routeBuilder.ProtectedPOSTWithMetadata(
	"/api-keys",
	"api-key-create",
	handler,
	limen.NewRouteMetadata(
		limen.WithRouteSummary("Create API key"),
		limen.WithRouteTags("api-keys"),
		limen.WithRouteResponse(http.StatusCreated, limen.OpenAPIResponse{
			Description: "API key created",
		}),
	),
)
```

Protected route helpers automatically mark routes as requiring session security.
Limen path parameters like `:provider` are emitted as OpenAPI `{provider}` path
parameters.
