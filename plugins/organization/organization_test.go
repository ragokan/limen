package organization

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

func TestCreateOrganizationCreatesOwnerMembership(t *testing.T) {
	t.Parallel()

	plugin, user := newTestOrganizationPlugin(t)

	org, err := plugin.CreateOrganization(context.Background(), user.ID, "Acme", "acme")
	require.NoError(t, err)
	assert.NotEmpty(t, org.ID)

	ok, err := plugin.HasRole(context.Background(), org.ID, user.ID, RoleOwner)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestListOrganizationsForUser(t *testing.T) {
	t.Parallel()

	plugin, user := newTestOrganizationPlugin(t)
	_, err := plugin.CreateOrganization(context.Background(), user.ID, "Acme", "acme")
	require.NoError(t, err)

	orgs, err := plugin.ListOrganizationsForUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Len(t, orgs, 1)
	assert.Equal(t, "acme", orgs[0].Slug)
}

func TestOrganizationRoleMiddleware(t *testing.T) {
	t.Parallel()

	plugin, owner, member := newTestOrganizationPluginWithUsers(t)
	org, err := plugin.CreateOrganization(context.Background(), owner.ID, "Acme", "acme")
	require.NoError(t, err)
	_, err = plugin.AddMember(context.Background(), org.ID, member.ID, RoleMember)
	require.NoError(t, err)

	allowed, err := plugin.HasRole(context.Background(), org.ID, owner.ID, RoleAdmin)
	require.NoError(t, err)
	assert.True(t, allowed, "owners satisfy admin checks")

	allowed, err = plugin.HasRole(context.Background(), org.ID, member.ID, RoleAdmin)
	require.NoError(t, err)
	assert.False(t, allowed)
}

func TestOrganizationRejectsOwnerEscalationAndRemoval(t *testing.T) {
	t.Parallel()

	plugin, owner, member := newTestOrganizationPluginWithUsers(t)
	org, err := plugin.CreateOrganization(context.Background(), owner.ID, "Acme", "acme")
	require.NoError(t, err)

	_, err = plugin.AddMember(context.Background(), org.ID, member.ID, RoleOwner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role is invalid")

	err = plugin.RemoveMember(context.Background(), org.ID, owner.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owners cannot be removed")

	_, err = plugin.CreateInvitation(context.Background(), org.ID, member.Email, RoleOwner)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "role is invalid")
}

func TestOrganizationHTTPRoutes(t *testing.T) {
	t.Parallel()

	plugin := New()
	auth, _ := limen.NewTestLimen(t, plugin)
	owner := limen.SeedTestUser(t, auth, "owner@test.com")
	member := limen.SeedTestUser(t, auth, "member@test.com")
	session := limen.SeedTestSession(t, auth, owner.ID, owner.Email)

	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/organizations", strings.NewReader(`{"name":"Acme","slug":"acme"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(session.Cookie)
	createResp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(createResp, createReq)

	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var org Organization
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&org))
	orgID := parseID(toPathID(org.ID))

	addReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/organizations/"+toPathID(orgID)+"/members", strings.NewReader(fmt.Sprintf(`{"user_id":%s,"role":"admin"}`, toPathID(member.ID))))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.AddCookie(session.Cookie)
	addResp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(addResp, addReq)

	require.Equal(t, http.StatusCreated, addResp.Code, addResp.Body.String())
	ok, err := plugin.HasRole(context.Background(), orgID, member.ID, RoleAdmin)
	require.NoError(t, err)
	assert.True(t, ok)

	listReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/auth/organizations", http.NoBody)
	listReq.AddCookie(session.Cookie)
	listResp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(listResp, listReq)
	require.Equal(t, http.StatusOK, listResp.Code, listResp.Body.String())

	removeReq := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/auth/organizations/"+toPathID(orgID)+"/members/"+toPathID(member.ID), http.NoBody)
	removeReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	removeReq.AddCookie(session.Cookie)
	removeResp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(removeResp, removeReq)

	require.Equal(t, http.StatusNoContent, removeResp.Code, removeResp.Body.String())
	ok, err = plugin.HasRole(context.Background(), orgID, member.ID)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestOrganizationHTTPManagementRequiresOwner(t *testing.T) {
	t.Parallel()

	plugin := New()
	auth, _ := limen.NewTestLimen(t, plugin)
	owner := limen.SeedTestUser(t, auth, "owner@test.com")
	admin := limen.SeedTestUser(t, auth, "admin@test.com")
	member := limen.SeedTestUser(t, auth, "member@test.com")
	adminSession := limen.SeedTestSession(t, auth, admin.ID, admin.Email)
	org, err := plugin.CreateOrganization(context.Background(), owner.ID, "Acme", "acme")
	require.NoError(t, err)
	_, err = plugin.AddMember(context.Background(), org.ID, admin.ID, RoleAdmin)
	require.NoError(t, err)

	addReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/organizations/"+toPathID(org.ID)+"/members", strings.NewReader(fmt.Sprintf(`{"user_id":%s,"role":"member"}`, toPathID(member.ID))))
	addReq.Header.Set("Content-Type", "application/json")
	addReq.AddCookie(adminSession.Cookie)
	addResp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(addResp, addReq)

	assert.Equal(t, http.StatusForbidden, addResp.Code)
}

func TestOrganizationInvitations(t *testing.T) {
	t.Parallel()

	plugin, owner, member := newTestOrganizationPluginWithUsers(t)
	org, err := plugin.CreateOrganization(context.Background(), owner.ID, "Acme", "acme")
	require.NoError(t, err)

	invitation, err := plugin.CreateInvitation(context.Background(), org.ID, member.Email, RoleAdmin)
	require.NoError(t, err)
	require.NotEmpty(t, invitation.Token)

	membership, err := plugin.AcceptInvitation(context.Background(), member.ID, invitation.Token)
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, membership.Role)

	_, err = plugin.AcceptInvitation(context.Background(), member.ID, invitation.Token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

func TestOrganizationInvitationRejectsWrongEmailAndRedactsToken(t *testing.T) {
	t.Parallel()

	plugin := New()
	auth, _ := limen.NewTestLimen(t, plugin)
	owner := limen.SeedTestUser(t, auth, "owner@test.com")
	member := limen.SeedTestUser(t, auth, "member@test.com")
	other := limen.SeedTestUser(t, auth, "other@test.com")
	org, err := plugin.CreateOrganization(context.Background(), owner.ID, "Acme", "acme")
	require.NoError(t, err)

	invitation, err := plugin.CreateInvitation(context.Background(), org.ID, member.Email, RoleMember)
	require.NoError(t, err)
	require.NotEmpty(t, invitation.Token)

	listed, err := plugin.ListInvitations(context.Background(), org.ID)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Empty(t, listed[0].Token)
	assert.NotContains(t, fmt.Sprint(listed[0].Raw()), invitation.Token)

	_, err = plugin.AcceptInvitation(context.Background(), other.ID, invitation.Token)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")
}

func TestOrganizationInvitationHTTPRoutes(t *testing.T) {
	t.Parallel()

	plugin := New()
	auth, _ := limen.NewTestLimen(t, plugin)
	owner := limen.SeedTestUser(t, auth, "owner@test.com")
	member := limen.SeedTestUser(t, auth, "member@test.com")
	ownerSession := limen.SeedTestSession(t, auth, owner.ID, owner.Email)
	memberSession := limen.SeedTestSession(t, auth, member.ID, member.Email)
	org, err := plugin.CreateOrganization(context.Background(), owner.ID, "Acme", "acme")
	require.NoError(t, err)

	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/organizations/"+toPathID(org.ID)+"/invitations", strings.NewReader(`{"email":"member@test.com","role":"member"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(ownerSession.Cookie)
	createResp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(createResp, createReq)

	require.Equal(t, http.StatusCreated, createResp.Code, createResp.Body.String())
	var invitation Invitation
	require.NoError(t, json.NewDecoder(createResp.Body).Decode(&invitation))
	require.NotEmpty(t, invitation.Token)

	acceptReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/organizations/invitations/accept", strings.NewReader(fmt.Sprintf(`{"token":%q}`, invitation.Token)))
	acceptReq.Header.Set("Content-Type", "application/json")
	acceptReq.AddCookie(memberSession.Cookie)
	acceptResp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(acceptResp, acceptReq)

	require.Equal(t, http.StatusOK, acceptResp.Code, acceptResp.Body.String())
	ok, err := plugin.HasRole(context.Background(), org.ID, member.ID, RoleMember)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestOrganizationOpenAPIMetadata(t *testing.T) {
	t.Parallel()

	plugin := New()
	auth, _ := limen.NewTestLimen(t, plugin)
	doc := auth.OpenAPI()

	create := requireOpenAPIOperation(t, doc, "/auth/organizations", "post")
	require.NotNil(t, create.RequestBody)
	assert.Contains(t, create.RequestBody.Content, "application/json")
	assert.Equal(t, "Created", create.Responses["201"].Description)

	addMember := requireOpenAPIOperation(t, doc, "/auth/organizations/{id}/members", "post")
	require.NotNil(t, addMember.RequestBody)
	assert.Contains(t, addMember.RequestBody.Content, "application/json")
	assert.Equal(t, "Created", addMember.Responses["201"].Description)

	removeMember := requireOpenAPIOperation(t, doc, "/auth/organizations/{id}/members/{user_id}", "delete")
	assert.Equal(t, "No Content", removeMember.Responses["204"].Description)
}

func newTestOrganizationPlugin(t *testing.T) (*organizationPlugin, *limen.User) {
	t.Helper()

	plugin := New()
	auth, _ := limen.NewTestLimen(t, plugin)
	user := limen.SeedTestUser(t, auth, "owner@test.com")
	return plugin, user
}

func newTestOrganizationPluginWithUsers(t *testing.T) (*organizationPlugin, *limen.User, *limen.User) {
	t.Helper()

	plugin := New()
	auth, _ := limen.NewTestLimen(t, plugin)
	owner := limen.SeedTestUser(t, auth, "owner@test.com")
	member := limen.SeedTestUser(t, auth, "member@test.com")
	return plugin, owner, member
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
