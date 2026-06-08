package credentialpassword

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

func TestOpenAPIMetadata(t *testing.T) {
	t.Parallel()

	auth, _ := limen.NewTestLimen(t, New(WithUsernameSupport(true)))
	doc := auth.OpenAPI()

	signIn := requireOpenAPIOperation(t, doc, "/auth/signin/credential", "post")
	assert.Equal(t, "signin", signIn.OperationID)
	assert.Equal(t, "Sign in with credential and password", signIn.Summary)
	assert.Equal(t, []string{limen.OpenAPIAuthTag}, signIn.Tags)
	assertOpenAPIRequestSchemaRef(t, signIn, limen.OpenAPIAuthCredentialSignInRequestSchema)
	assertOpenAPIResponseSchemaRef(t, signIn, http.StatusOK, limen.OpenAPIAuthSessionResponseSchema)

	setPassword := requireOpenAPIOperation(t, doc, "/auth/passwords", "put")
	assert.Equal(t, "Set password", setPassword.Summary)
	assertOpenAPIRequestSchemaRef(t, setPassword, limen.OpenAPIAuthPasswordSetRequestSchema)
	assertOpenAPIResponseSchemaRef(t, setPassword, http.StatusOK, limen.OpenAPIAuthSessionResponseSchema)
	require.Len(t, setPassword.Security, 1)
	assert.Contains(t, setPassword.Security[0], "sessionCookie")

	usernameCheck := requireOpenAPIOperation(t, doc, "/auth/usernames/check", "post")
	assert.Equal(t, "Check username availability", usernameCheck.Summary)
	assertOpenAPIRequestSchemaRef(t, usernameCheck, limen.OpenAPIAuthUsernameCheckRequestSchema)
	assertOpenAPIResponseSchemaRef(t, usernameCheck, http.StatusOK, limen.OpenAPIAuthUsernameAvailabilityResponseSchema)

	require.Contains(t, doc.Components.Schemas, limen.OpenAPIAuthCredentialSignInRequestSchema)
	require.Contains(t, doc.Components.Schemas, limen.OpenAPIAuthPasswordSetRequestSchema)
	require.Contains(t, doc.Components.Schemas, limen.OpenAPIAuthUsernameAvailabilityResponseSchema)
}

func TestOpenAPIOmitsUsernameCheckWhenUsernameSupportDisabled(t *testing.T) {
	t.Parallel()

	auth, _ := limen.NewTestLimen(t, New())
	doc := auth.OpenAPI()

	assert.NotContains(t, doc.Paths, "/auth/usernames/check")
	assert.NotContains(t, doc.Components.Schemas, limen.OpenAPIAuthUsernameCheckRequestSchema)
	assert.NotContains(t, doc.Components.Schemas, limen.OpenAPIAuthUsernameAvailabilityResponseSchema)
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
