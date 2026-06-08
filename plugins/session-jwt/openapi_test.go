package sessionjwt

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

func TestOpenAPIMetadataForRefreshRoute(t *testing.T) {
	t.Parallel()

	auth, _ := limen.NewTestLimen(t, New())
	doc := auth.OpenAPI()

	refresh := requireOpenAPIOperation(t, doc, "/auth/refresh", "post")
	assert.Equal(t, "session-jwt-refresh", refresh.OperationID)
	assert.Equal(t, "Refresh access token", refresh.Summary)
	assert.Equal(t, []string{limen.OpenAPIAuthTag}, refresh.Tags)
	assertOpenAPIRequestSchemaRef(t, refresh, limen.OpenAPIAuthRefreshRequestSchema)
	assertOpenAPIResponseSchemaRef(t, refresh, http.StatusOK, limen.OpenAPIAuthSessionResponseSchema)

	me := requireOpenAPIOperation(t, doc, "/auth/me", "get")
	require.Contains(t, doc.Components.SecuritySchemes, "bearerAuth")
	assert.Equal(t, "JWT", doc.Components.SecuritySchemes["bearerAuth"].BearerFormat)
	assert.NotContains(t, doc.Components.SecuritySchemes, "sessionCookie")
	require.Len(t, me.Security, 1)
	assert.Contains(t, me.Security[0], "bearerAuth")

	require.Contains(t, doc.Components.Schemas, limen.OpenAPIAuthRefreshRequestSchema)
	require.Contains(t, doc.Components.Schemas, limen.OpenAPIAuthSessionResponseSchema)
}

func TestOpenAPIOmitsRefreshRouteWhenRefreshTokensDisabled(t *testing.T) {
	t.Parallel()

	auth, _ := limen.NewTestLimen(t, New(WithRefreshToken(false)))
	doc := auth.OpenAPI()

	assert.NotContains(t, doc.Paths, "/auth/refresh")
	assert.NotContains(t, doc.Components.Schemas, limen.OpenAPIAuthRefreshRequestSchema)
}

func requireOpenAPIOperation(t *testing.T, doc *limen.OpenAPIDocument, path string, method string) limen.OpenAPIOperation {
	t.Helper()

	pathItem, ok := doc.Paths[path]
	require.True(t, ok, "missing OpenAPI path %s", path)
	operation, ok := pathItem[method]
	require.True(t, ok, "missing OpenAPI operation %s %s", method, path)
	return operation
}

func assertOpenAPIRequestSchemaRef(t *testing.T, operation limen.OpenAPIOperation, schemaName string) {
	t.Helper()

	require.NotNil(t, operation.RequestBody)
	media, ok := operation.RequestBody.Content["application/json"]
	require.True(t, ok, "missing JSON request body content")
	assert.Equal(t, limen.OpenAPIRefSchema(schemaName), media.Schema)
}

func assertOpenAPIResponseSchemaRef(t *testing.T, operation limen.OpenAPIOperation, status int, schemaName string) {
	t.Helper()

	response, ok := operation.Responses[strconv.Itoa(status)]
	require.True(t, ok, "missing OpenAPI response %d", status)
	media, ok := response.Content["application/json"]
	require.True(t, ok, "missing JSON response content")
	assert.Equal(t, limen.OpenAPIRefSchema(schemaName), media.Schema)
}
