package apikey

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

func TestCreateAndValidateAPIKey(t *testing.T) {
	t.Parallel()

	plugin, user := newTestAPIKeyPlugin(t, WithAllowedScopes("read", "write"))
	created, err := plugin.CreateAPIKey(context.Background(), user.ID, "CLI", WithScopes("read", "write", "read"))
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(created.Key, defaultKeyPrefix))
	assert.NotEmpty(t, created.APIKey.Prefix)
	assert.NotEmpty(t, created.APIKey.keyHash)
	assert.NotEqual(t, created.Key, created.APIKey.keyHash)
	assert.Equal(t, []string{"read", "write"}, created.APIKey.Scopes)

	validated, err := plugin.ValidateAPIKey(context.Background(), created.Key, "read")
	require.NoError(t, err)
	assert.Equal(t, created.APIKey.ID, validated.ID)
	require.NotNil(t, validated.LastUsedAt)

	_, err = plugin.ValidateAPIKey(context.Background(), created.Key, "admin")
	require.ErrorIs(t, err, ErrAPIKeyScope)
}

func TestCreateAPIKeyRejectsUnallowedOrInvalidScopes(t *testing.T) {
	t.Parallel()

	plugin, user := newTestAPIKeyPlugin(t, WithAllowedScopes("read"))

	_, err := plugin.CreateAPIKey(context.Background(), user.ID, "Bad", WithScopes("admin"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	_, err = plugin.CreateAPIKey(context.Background(), user.ID, "Comma", WithScopes("read,admin"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestValidateAPIKeyRejectsExpiredAndRevokedKeys(t *testing.T) {
	t.Parallel()

	plugin, user := newTestAPIKeyPlugin(t)
	expiredAt := time.Now().Add(-time.Hour)
	expired, err := plugin.CreateAPIKey(context.Background(), user.ID, "Expired", WithExpiresAt(expiredAt))
	require.NoError(t, err)

	_, err = plugin.ValidateAPIKey(context.Background(), expired.Key)
	require.ErrorIs(t, err, ErrAPIKeyExpired)

	active, err := plugin.CreateAPIKey(context.Background(), user.ID, "Active")
	require.NoError(t, err)
	require.NoError(t, plugin.RevokeAPIKey(context.Background(), user.ID, active.APIKey.ID))

	_, err = plugin.ValidateAPIKey(context.Background(), active.Key)
	require.ErrorIs(t, err, ErrAPIKeyRevoked)
}

func TestAPIKeyMiddlewareStoresValidatedKeyInContext(t *testing.T) {
	t.Parallel()

	plugin, user := newTestAPIKeyPlugin(t, WithAllowedScopes("jobs:write"))
	created, err := plugin.CreateAPIKey(context.Background(), user.ID, "Service", WithScopes("jobs:write"))
	require.NoError(t, err)

	handler := plugin.MiddlewareRequireAPIKey("jobs:write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := GetAPIKeyFromContext(r.Context())
		require.True(t, ok)
		assert.Equal(t, created.APIKey.ID, key.ID)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/jobs", http.NoBody)
	req.Header.Set(defaultHeaderName, created.Key)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestAPIKeyMiddlewareRejectsMissingAndWrongScope(t *testing.T) {
	t.Parallel()

	plugin, user := newTestAPIKeyPlugin(t, WithAllowedScopes("jobs:read", "jobs:write"))
	created, err := plugin.CreateAPIKey(context.Background(), user.ID, "Service", WithScopes("jobs:read"))
	require.NoError(t, err)

	handler := plugin.MiddlewareRequireAPIKey("jobs:write")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	missingReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/jobs", http.NoBody)
	missingResp := httptest.NewRecorder()
	handler.ServeHTTP(missingResp, missingReq)
	assert.Equal(t, http.StatusUnauthorized, missingResp.Code)

	wrongScopeReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/jobs", http.NoBody)
	wrongScopeReq.Header.Set(defaultHeaderName, created.Key)
	wrongScopeResp := httptest.NewRecorder()
	handler.ServeHTTP(wrongScopeResp, wrongScopeReq)
	assert.Equal(t, http.StatusForbidden, wrongScopeResp.Code)
}

func TestAPIKeyHTTPRoutes(t *testing.T) {
	t.Parallel()

	plugin := New(WithAllowedScopes("read"))
	auth, _ := limen.NewTestLimen(t, plugin)
	user := limen.SeedTestUser(t, auth, "api-http@test.com")
	session := limen.SeedTestSession(t, auth, user.ID, user.Email)

	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/api-keys", strings.NewReader(`{"name":"CLI","scopes":["read"]}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(session.Cookie)
	createResp := httptest.NewRecorder()

	auth.Handler().ServeHTTP(createResp, createReq)

	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	assert.NotContains(t, createResp.Body.String(), "key_hash")
	var created CreatedAPIKey
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&created))
	require.NotEmpty(t, created.Key)
	require.NotNil(t, created.APIKey)

	listReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/api-keys", http.NoBody)
	listReq.AddCookie(session.Cookie)
	listResp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(listResp, listReq)

	require.Equal(t, http.StatusOK, listResp.Code, listResp.Body.String())
	assert.NotContains(t, listResp.Body.String(), "key_hash")

	revokeReq := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/auth/api-keys/"+toPathID(created.APIKey.ID), http.NoBody)
	revokeReq.AddCookie(session.Cookie)
	revokeReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	revokeResp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(revokeResp, revokeReq)

	require.Equal(t, http.StatusNoContent, revokeResp.Code, revokeResp.Body.String())
	_, err := plugin.ValidateAPIKey(context.Background(), created.Key)
	require.True(t, errors.Is(err, ErrAPIKeyRevoked), "got %v", err)
}

func TestAPIKeyOpenAPIMetadata(t *testing.T) {
	t.Parallel()

	plugin := New(WithAllowedScopes("read"))
	auth, _ := limen.NewTestLimen(t, plugin)
	doc := auth.OpenAPI()

	create := requireOpenAPIOperation(t, doc, "/auth/api-keys", "post")
	require.NotNil(t, create.RequestBody)
	assert.Contains(t, create.RequestBody.Content, "application/json")
	assert.Equal(t, "Created", create.Responses["201"].Description)

	revoke := requireOpenAPIOperation(t, doc, "/auth/api-keys/{id}", "delete")
	assert.Equal(t, "No Content", revoke.Responses["204"].Description)
}

func newTestAPIKeyPlugin(t *testing.T, opts ...ConfigOption) (*apiKeyPlugin, *limen.User) {
	t.Helper()

	plugin := New(opts...)
	auth, _ := limen.NewTestLimen(t, plugin)
	user := limen.SeedTestUser(t, auth, "api-key@test.com")
	return plugin, user
}

func requireOpenAPIOperation(t *testing.T, doc *limen.OpenAPIDocument, path string, method string) limen.OpenAPIOperation {
	t.Helper()

	pathItem, ok := doc.Paths[path]
	require.True(t, ok, "missing OpenAPI path %s", path)
	operation, ok := pathItem[method]
	require.True(t, ok, "missing OpenAPI operation %s %s", method, path)
	return operation
}

func toPathID(id any) string {
	switch typed := id.(type) {
	case string:
		return typed
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	default:
		return fmt.Sprint(id)
	}
}
