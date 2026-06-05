package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

func TestIsAdminUsesConfiguredEmail(t *testing.T) {
	t.Parallel()

	plugin := New(WithAdminEmails("admin@test.com"))
	auth, _ := limen.NewTestLimen(t, plugin)
	adminUser := limen.SeedTestUser(t, auth, "admin@test.com")
	normalUser := limen.SeedTestUser(t, auth, "user@test.com")

	assert.False(t, plugin.IsAdmin(adminUser), "email admin requires a verified email")
	verifyAdminEmail(t, plugin, adminUser.ID)
	refreshed, err := plugin.core.DBAction.FindUserByID(context.Background(), adminUser.ID)
	require.NoError(t, err)
	assert.True(t, plugin.IsAdmin(refreshed))
	assert.False(t, plugin.IsAdmin(normalUser))
}

func TestAdminHTTPRoutesRequireAdmin(t *testing.T) {
	t.Parallel()

	plugin := New(WithAdminUserIDs(999))
	auth, _ := limen.NewTestLimen(t, plugin)
	normalUser := limen.SeedTestUser(t, auth, "user@test.com")
	normalSession := limen.SeedTestSession(t, auth, normalUser.ID, normalUser.Email)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/admin/users", http.NoBody)
	req.AddCookie(normalSession.Cookie)
	w := httptest.NewRecorder()

	auth.Handler().ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminCanListUsersWithoutPassword(t *testing.T) {
	t.Parallel()

	plugin := New(WithAdminUserIDs(1))
	auth, _ := limen.NewTestLimen(t, plugin)
	adminUser := limen.SeedTestUser(t, auth, "admin@test.com")
	limen.SeedTestUser(t, auth, "user@test.com")
	adminSession := limen.SeedTestSession(t, auth, adminUser.ID, adminUser.Email)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/admin/users", http.NoBody)
	req.AddCookie(adminSession.Cookie)
	w := httptest.NewRecorder()

	auth.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.NotContains(t, w.Body.String(), "password")

	var users []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&users))
	assert.Len(t, users, 2)
}

func TestAdminCanRevokeUserSessions(t *testing.T) {
	t.Parallel()

	plugin := New(WithAdminUserIDs(1))
	auth, _ := limen.NewTestLimen(t, plugin)
	adminUser := limen.SeedTestUser(t, auth, "admin@test.com")
	normalUser := limen.SeedTestUser(t, auth, "user@test.com")
	adminSession := limen.SeedTestSession(t, auth, adminUser.ID, adminUser.Email)
	limen.SeedTestSession(t, auth, normalUser.ID, normalUser.Email)

	sessions, err := auth.ListSessions(context.Background(), normalUser.ID)
	require.NoError(t, err)
	require.Len(t, sessions, 1)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/admin/users/"+toPathID(normalUser.ID)+"/revoke-sessions", http.NoBody)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.AddCookie(adminSession.Cookie)
	w := httptest.NewRecorder()

	auth.Handler().ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
	sessions, err = auth.ListSessions(context.Background(), normalUser.ID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestAdminOpenAPIMetadata(t *testing.T) {
	t.Parallel()

	plugin := New(WithAdminUserIDs(1))
	auth, _ := limen.NewTestLimen(t, plugin)
	doc := auth.OpenAPI()

	revoke := requireOpenAPIOperation(t, doc, "/auth/admin/users/{id}/revoke-sessions", "post")
	assert.Equal(t, "No Content", revoke.Responses["204"].Description)
}

func verifyAdminEmail(t *testing.T, plugin *adminPlugin, userID any) {
	t.Helper()

	now := time.Now()
	require.NoError(t, plugin.core.Update(context.Background(), plugin.core.Schema.User, &limen.User{
		EmailVerifiedAt: &now,
	}, []limen.Where{limen.Eq(plugin.core.Schema.User.GetIDField(), userID)}))
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
