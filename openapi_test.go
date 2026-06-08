package limen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type openAPITestPlugin struct{}

func (p openAPITestPlugin) Name() PluginName {
	return "openapi-test"
}

func (p openAPITestPlugin) Initialize(core *LimenCore) error {
	return nil
}

func (p openAPITestPlugin) PluginHTTPConfig() PluginHTTPConfig {
	return PluginHTTPConfig{BasePath: "/test"}
}

func (p openAPITestPlugin) RegisterRoutes(httpCore *LimenHTTPCore, routeBuilder *RouteBuilder) {
	routeBuilder.GETWithMetadata(
		"/:provider/items/:id",
		"test-route",
		func(w http.ResponseWriter, r *http.Request) {},
		NewRouteMetadata(
			WithRouteSummary("Fetch item"),
			WithRouteTags("test"),
			WithRouteParameters(OpenAPIParameter{
				Name:   "verbose",
				In:     "query",
				Schema: OpenAPIStringSchema(),
			}),
		),
	)
	routeBuilder.ProtectedPOSTWithMetadata(
		"/secure",
		"secure-route",
		func(w http.ResponseWriter, r *http.Request) {},
		NewRouteMetadata(
			WithRouteAllowedContentTypes("application/x-www-form-urlencoded"),
			WithRouteResponse(http.StatusCreated, OpenAPIResponse{Description: "Created"}),
		),
	)
}

type openAPIRateLimitTestPlugin struct{}

func (p openAPIRateLimitTestPlugin) Name() PluginName {
	return "openapi-rate-limit-test"
}

func (p openAPIRateLimitTestPlugin) Initialize(core *LimenCore) error {
	return nil
}

func (p openAPIRateLimitTestPlugin) PluginHTTPConfig() PluginHTTPConfig {
	return PluginHTTPConfig{
		BasePath:       "/limited",
		RateLimitRules: []*RateLimitRule{NewRateLimitRule("/thing", 100, time.Hour)},
	}
}

func (p openAPIRateLimitTestPlugin) RegisterRoutes(httpCore *LimenHTTPCore, routeBuilder *RouteBuilder) {
	routeBuilder.GET("/thing", "openapi-rate-limited-thing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

func TestOpenAPIIncludesCoreAndPluginRoutes(t *testing.T) {
	t.Parallel()

	l := newOpenAPITestLimen(t)

	doc := l.OpenAPI(
		WithOpenAPITitle("Custom Auth API"),
		WithOpenAPIVersion("2026.1"),
		WithOpenAPIDescription("Generated test spec"),
	)

	require.Equal(t, openAPIVersion, doc.OpenAPI)
	assert.Equal(t, "Custom Auth API", doc.Info.Title)
	assert.Equal(t, "2026.1", doc.Info.Version)
	assert.Equal(t, "Generated test spec", doc.Info.Description)
	require.Len(t, doc.Servers, 1)
	assert.Equal(t, "https://api.example.test", doc.Servers[0].URL)

	me := requireOperation(t, doc, "/api/auth/me", "get")
	assert.Equal(t, "me", me.OperationID)
	assert.Equal(t, []string{OpenAPIAuthTag}, me.Tags)
	require.Len(t, me.Security, 1)
	assert.Contains(t, me.Security[0], openAPISessionCookieScheme)
	assertResponseSchemaRef(t, me, http.StatusOK, OpenAPIAuthSessionResponseSchema)
	assertResponseSchemaRef(t, me, http.StatusUnauthorized, OpenAPIAuthErrorResponseSchema)
	assert.Equal(t, "apiKey", doc.Components.SecuritySchemes[openAPISessionCookieScheme].Type)
	require.Contains(t, doc.Components.Schemas, OpenAPIAuthSessionResponseSchema)
	require.Contains(t, doc.Components.Schemas, OpenAPIAuthUserSchema)
	require.Contains(t, doc.Components.Schemas, OpenAPIAuthTokensSchema)
	require.Contains(t, doc.Components.Schemas, OpenAPIAuthErrorResponseSchema)
	require.Contains(t, doc.Components.Schemas, OpenAPIAuthSessionListResponseSchema)
	assert.NotContains(t, doc.Components.Schemas, OpenAPIAuthCredentialSignInRequestSchema)

	sessionSchema := doc.Components.Schemas[OpenAPIAuthSessionResponseSchema]
	sessionProperties, ok := sessionSchema["properties"].(map[string]OpenAPISchema)
	require.True(t, ok)
	assert.Equal(t, OpenAPIRefSchema(OpenAPIAuthTokensSchema), sessionProperties["tokens"])

	errorSchema := doc.Components.Schemas[OpenAPIAuthErrorResponseSchema]
	errorProperties, ok := errorSchema["properties"].(map[string]OpenAPISchema)
	require.True(t, ok)
	assert.Equal(t, OpenAPIStringSchema(), errorProperties["message"])

	signout := requireOperation(t, doc, "/api/auth/signout", "post")
	assert.Equal(t, "No Content", signout.Responses["204"].Description)
	assertResponseSchemaRef(t, signout, http.StatusUnauthorized, OpenAPIAuthErrorResponseSchema)

	pluginRoute := requireOperation(t, doc, "/api/auth/test/{provider}/items/{id}", "get")
	assert.Equal(t, "test-route", pluginRoute.OperationID)
	assert.Equal(t, "Fetch item", pluginRoute.Summary)
	assert.Equal(t, []string{"test"}, pluginRoute.Tags)
	assertParameter(t, pluginRoute.Parameters, "path", "provider", true)
	assertParameter(t, pluginRoute.Parameters, "path", "id", true)
	assertParameter(t, pluginRoute.Parameters, "query", "verbose", false)

	secure := requireOperation(t, doc, "/api/auth/test/secure", "post")
	require.Len(t, secure.Security, 1)
	assert.Contains(t, secure.Security[0], openAPISessionCookieScheme)
	require.NotNil(t, secure.RequestBody)
	assert.Contains(t, secure.RequestBody.Content, "application/x-www-form-urlencoded")
	assert.Equal(t, "Created", secure.Responses["201"].Description)
	assertResponseSchemaRef(t, secure, http.StatusUnprocessableEntity, OpenAPIAuthErrorResponseSchema)
}

func TestOpenAPIIncludesBearerAlternativeWhenEnabled(t *testing.T) {
	t.Parallel()

	l, err := New(&Config{
		BaseURL:  "https://api.example.test",
		Database: newTestMemoryAdapter(t),
		Secret:   testSecret,
		HTTP:     NewDefaultHTTPConfig(WithHTTPBasePath("/api/auth")),
		Session:  NewDefaultSessionConfig(WithBearerEnabled()),
		Plugins:  []Plugin{openAPITestPlugin{}},
	})
	require.NoError(t, err)

	doc := l.OpenAPI()
	me := requireOperation(t, doc, "/api/auth/me", "get")

	require.Contains(t, doc.Components.SecuritySchemes, openAPISessionCookieScheme)
	require.Contains(t, doc.Components.SecuritySchemes, openAPIBearerSessionScheme)
	require.Len(t, me.Security, 2)
	assert.Equal(t, OpenAPISecurityRequirement{openAPISessionCookieScheme: []string{}}, me.Security[0])
	assert.Equal(t, OpenAPISecurityRequirement{openAPIBearerSessionScheme: []string{}}, me.Security[1])
}

func TestOpenAPIDoesNotMutateHTTPConfigOrConsumeRateLimitOverrides(t *testing.T) {
	t.Parallel()

	auth, err := New(&Config{
		BaseURL:  "https://api.example.test",
		Database: newTestMemoryAdapter(t),
		Secret:   testSecret,
		HTTP: NewDefaultHTTPConfig(
			WithHTTPBasePath("api/auth"),
			WithHTTPRateLimiter(WithRateLimiterCustomRule("/thing", 1, time.Hour)),
		),
		Plugins: []Plugin{openAPIRateLimitTestPlugin{}},
	})
	require.NoError(t, err)

	doc := auth.OpenAPI()
	requireOperation(t, doc, "/api/auth/limited/thing", "get")

	assert.Equal(t, "api/auth", auth.config.HTTP.basePath)
	rule, ok := auth.config.HTTP.rateLimiter.customRules["/thing"]
	require.True(t, ok)
	assert.Equal(t, "/thing", rule.path)
	assert.Nil(t, rule.pathRegex)

	handler := auth.Handler()
	req1 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/limited/thing", http.NoBody)
	req1.RemoteAddr = "203.0.113.99:1234"
	resp1 := httptest.NewRecorder()
	handler.ServeHTTP(resp1, req1)
	require.Equal(t, http.StatusNoContent, resp1.Code, resp1.Body.String())

	req2 := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/limited/thing", http.NoBody)
	req2.RemoteAddr = "203.0.113.99:1234"
	resp2 := httptest.NewRecorder()
	handler.ServeHTTP(resp2, req2)
	assert.Equal(t, http.StatusTooManyRequests, resp2.Code, resp2.Body.String())
}

func TestOpenAPIHonorsDisabledRoutes(t *testing.T) {
	t.Parallel()

	l := newOpenAPITestLimen(t, WithHTTPDisabledPaths([]string{"test-route"}))

	doc := l.OpenAPI()
	assert.NotContains(t, doc.Paths, "/api/auth/test/{provider}/items/{id}")
	requireOperation(t, doc, "/api/auth/test/secure", "post")
}

func TestOpenAPIHonorsPluginHTTPOverrides(t *testing.T) {
	t.Parallel()

	l := newOpenAPITestLimen(t, WithHTTPOverrides(map[string]*PluginHTTPOverride{
		"openapi-test": {BasePath: "/overridden"},
	}))

	doc := l.OpenAPI()
	assert.NotContains(t, doc.Paths, "/api/auth/test/{provider}/items/{id}")
	requireOperation(t, doc, "/api/auth/overridden/{provider}/items/{id}", "get")
}

func TestOpenAPIHandlerWritesJSON(t *testing.T) {
	t.Parallel()

	l := newOpenAPITestLimen(t)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/openapi.json", http.NoBody)
	w := httptest.NewRecorder()

	l.OpenAPIHandler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var doc OpenAPIDocument
	require.NoError(t, json.NewDecoder(w.Body).Decode(&doc))
	requireOperation(t, &doc, "/api/auth/me", "get")
}

func TestHTTPHandlerCanServeOpenAPI(t *testing.T) {
	t.Parallel()

	l := newOpenAPITestLimen(t, WithHTTPOpenAPI("/openapi.json"))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/auth/openapi.json", http.NoBody)
	w := httptest.NewRecorder()

	l.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var doc OpenAPIDocument
	require.NoError(t, json.NewDecoder(w.Body).Decode(&doc))
	requireOperation(t, &doc, "/api/auth/me", "get")
	assert.NotContains(t, doc.Paths, "/api/auth/openapi.json")
}

func newOpenAPITestLimen(t *testing.T, httpOpts ...HTTPConfigOption) *Limen {
	t.Helper()

	l, err := New(&Config{
		BaseURL:  "https://api.example.test",
		Database: newTestMemoryAdapter(t),
		Secret:   testSecret,
		HTTP:     NewDefaultHTTPConfig(append([]HTTPConfigOption{WithHTTPBasePath("/api/auth")}, httpOpts...)...),
		Plugins:  []Plugin{openAPITestPlugin{}},
	})
	require.NoError(t, err)
	return l
}

func requireOperation(t *testing.T, doc *OpenAPIDocument, path string, method string) OpenAPIOperation {
	t.Helper()

	pathItem, ok := doc.Paths[path]
	require.True(t, ok, "missing OpenAPI path %s", path)
	operation, ok := pathItem[method]
	require.True(t, ok, "missing OpenAPI operation %s %s", method, path)
	return operation
}

func assertParameter(t *testing.T, parameters []OpenAPIParameter, in string, name string, required bool) {
	t.Helper()

	for _, parameter := range parameters {
		if parameter.In == in && parameter.Name == name {
			assert.Equal(t, required, parameter.Required)
			return
		}
	}
	t.Fatalf("missing OpenAPI parameter %s:%s", in, name)
}

func assertResponseSchemaRef(t *testing.T, operation OpenAPIOperation, status int, schemaName string) {
	t.Helper()

	response, ok := operation.Responses[strconv.Itoa(status)]
	require.True(t, ok, "missing OpenAPI response %d", status)
	media, ok := response.Content[defaultOpenAPIContentType]
	require.True(t, ok, "missing JSON response content")
	assert.Equal(t, OpenAPIRefSchema(schemaName), media.Schema)
}
