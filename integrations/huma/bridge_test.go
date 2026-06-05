package limenhuma

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

type bridgeTestPlugin struct{}

func (p bridgeTestPlugin) Name() limen.PluginName {
	return "huma-bridge-test"
}

func (p bridgeTestPlugin) Initialize(core *limen.LimenCore) error {
	return nil
}

func (p bridgeTestPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{BasePath: "/bridge"}
}

func (p bridgeTestPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	routeBuilder.ProtectedPOSTWithMetadata(
		"/:id",
		"bridge-create",
		func(w http.ResponseWriter, r *http.Request) {},
		limen.NewRouteMetadata(
			limen.WithRouteSummary("Create bridge resource"),
			limen.WithRouteTags("bridge"),
			limen.WithRouteRequestBody(&limen.OpenAPIRequestBody{
				Required: true,
				Content: map[string]limen.OpenAPIMediaType{
					"application/json": {
						Schema: limen.OpenAPIObjectSchema(map[string]limen.OpenAPISchema{
							"name": limen.OpenAPIStringSchema(),
						}, "name"),
					},
				},
			}),
			limen.WithRouteResponse(http.StatusCreated, limen.OpenAPIResponse{Description: "Created"}),
		),
	)
}

func TestMergeAddsLimenSpecToHumaOpenAPI(t *testing.T) {
	t.Parallel()

	auth, _ := limen.NewTestLimen(t, bridgeTestPlugin{})
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Host API", "1.0.0"))

	require.NoError(t, Merge(api, auth))

	op := requireHumaOperation(t, api.OpenAPI(), "/auth/bridge/{id}", http.MethodPost)
	assert.Equal(t, "bridge-create", op.OperationID)
	assert.Equal(t, "Create bridge resource", op.Summary)
	assert.Equal(t, []string{"bridge"}, op.Tags)
	require.Len(t, op.Parameters, 1)
	assert.Equal(t, "id", op.Parameters[0].Name)
	assert.Equal(t, "path", op.Parameters[0].In)
	assert.True(t, op.Parameters[0].Required)

	require.NotNil(t, op.RequestBody)
	assert.True(t, op.RequestBody.Required)
	require.Contains(t, op.RequestBody.Content, "application/json")
	assert.Equal(t, "object", op.RequestBody.Content["application/json"].Schema.Type)
	assert.Equal(t, []string{"name"}, op.RequestBody.Content["application/json"].Schema.Required)

	require.Contains(t, op.Responses, "201")
	assert.Equal(t, "Created", op.Responses["201"].Description)
	require.NotNil(t, api.OpenAPI().Components)
	require.Contains(t, api.OpenAPI().Components.SecuritySchemes, "sessionCookie")
	require.Len(t, op.Security, 1)
	assert.Contains(t, op.Security[0], "sessionCookie")
}

func TestMergeReturnsDuplicateOperationError(t *testing.T) {
	t.Parallel()

	auth, _ := limen.NewTestLimen(t, bridgeTestPlugin{})
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Host API", "1.0.0"))
	api.OpenAPI().AddOperation(&huma.Operation{
		OperationID: "bridge-create",
		Method:      http.MethodGet,
		Path:        "/host/bridge",
		Responses: map[string]*huma.Response{
			"200": {Description: "OK"},
		},
	})

	err := Merge(api, auth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate operation ID")
}

func TestMergeRejectsExistingPathMethod(t *testing.T) {
	t.Parallel()

	auth, _ := limen.NewTestLimen(t, bridgeTestPlugin{})
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Host API", "1.0.0"))
	api.OpenAPI().AddOperation(&huma.Operation{
		OperationID: "host-create",
		Method:      http.MethodPost,
		Path:        "/auth/bridge/{id}",
		Responses: map[string]*huma.Response{
			"200": {Description: "OK"},
		},
	})

	err := Merge(api, auth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting operation")
}

func TestMergeRejectsConflictingSecurityScheme(t *testing.T) {
	t.Parallel()

	auth, _ := limen.NewTestLimen(t, bridgeTestPlugin{})
	mux := http.NewServeMux()
	api := humago.New(mux, huma.DefaultConfig("Host API", "1.0.0"))
	if api.OpenAPI().Components == nil {
		api.OpenAPI().Components = &huma.Components{}
	}
	if api.OpenAPI().Components.SecuritySchemes == nil {
		api.OpenAPI().Components.SecuritySchemes = make(map[string]*huma.SecurityScheme)
	}
	api.OpenAPI().Components.SecuritySchemes["sessionCookie"] = &huma.SecurityScheme{
		Type:   "http",
		Scheme: "bearer",
	}

	err := Merge(api, auth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting security scheme")
}

func requireHumaOperation(t *testing.T, doc *huma.OpenAPI, path string, method string) *huma.Operation {
	t.Helper()

	require.NotNil(t, doc)
	pathItem, ok := doc.Paths[path]
	require.True(t, ok, "missing path %s", path)
	switch method {
	case http.MethodGet:
		require.NotNil(t, pathItem.Get)
		return pathItem.Get
	case http.MethodPost:
		require.NotNil(t, pathItem.Post)
		return pathItem.Post
	default:
		t.Fatalf("unsupported method %s", method)
		return nil
	}
}
