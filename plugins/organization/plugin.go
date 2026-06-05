package organization

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ragokan/limen"
)

type organizationPlugin struct {
	core             *limen.LimenCore
	organization     *organizationSchema
	membership       *membershipSchema
	invitation       *invitationSchema
	defaultOwnerRole Role
	invitationTTL    time.Duration
}

const openAPIResponseCreated = "Created"

type API interface {
	CreateOrganization(ctx context.Context, userID any, name string, slug string) (*Organization, error)
	ListOrganizationsForUser(ctx context.Context, userID any) ([]*Organization, error)
	AddMember(ctx context.Context, organizationID any, userID any, role Role) (*Membership, error)
	RemoveMember(ctx context.Context, organizationID any, userID any) error
	GetMembership(ctx context.Context, organizationID any, userID any) (*Membership, error)
	HasRole(ctx context.Context, organizationID any, userID any, roles ...Role) (bool, error)
	CreateInvitation(ctx context.Context, organizationID any, email string, role Role) (*Invitation, error)
	ListInvitations(ctx context.Context, organizationID any) ([]*Invitation, error)
	AcceptInvitation(ctx context.Context, userID any, token string) (*Membership, error)
	MiddlewareRequireOrganizationRole(paramName string, roles ...Role) limen.Middleware
}

func New() *organizationPlugin {
	return &organizationPlugin{
		defaultOwnerRole: RoleOwner,
		invitationTTL:    7 * 24 * time.Hour,
	}
}

func Use(auth *limen.Limen) API {
	return limen.Use[API](auth, limen.PluginOrganization)
}

func (p *organizationPlugin) Name() limen.PluginName {
	return limen.PluginOrganization
}

func (p *organizationPlugin) Initialize(core *limen.LimenCore) error {
	p.core = core
	return nil
}

func (p *organizationPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: defaultBasePath,
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRule("", 60, time.Minute),
		},
	}
}

func (p *organizationPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	handlers := newHandlers(p, httpCore)
	routeBuilder.ProtectedPOSTWithMetadata("", "organization-create", handlers.Create, routeMetadata(
		"Create organization",
		limen.WithRouteAllowedContentTypes("application/json"),
		limen.WithRouteResponse(http.StatusCreated, limen.OpenAPIResponse{Description: openAPIResponseCreated}),
	))
	routeBuilder.ProtectedGETWithMetadata("", "organization-list", handlers.List, routeMetadata("List organizations"))
	routeBuilder.ProtectedPOSTWithMetadata("/:id/members", "organization-add-member", handlers.AddMember, routeMetadata(
		"Add organization member",
		limen.WithRouteAllowedContentTypes("application/json"),
		limen.WithRouteResponse(http.StatusCreated, limen.OpenAPIResponse{Description: openAPIResponseCreated}),
	), p.MiddlewareRequireOrganizationRole("id", RoleOwner))
	routeBuilder.ProtectedDELETEWithMetadata("/:id/members/:user_id", "organization-remove-member", handlers.RemoveMember, routeMetadata(
		"Remove organization member",
		limen.WithRouteResponse(http.StatusNoContent, limen.OpenAPIResponse{Description: "No Content"}),
	), p.MiddlewareRequireOrganizationRole("id", RoleOwner))
	routeBuilder.ProtectedPOSTWithMetadata("/:id/invitations", "organization-create-invitation", handlers.CreateInvitation, routeMetadata(
		"Create organization invitation",
		limen.WithRouteAllowedContentTypes("application/json"),
		limen.WithRouteResponse(http.StatusCreated, limen.OpenAPIResponse{Description: openAPIResponseCreated}),
	), p.MiddlewareRequireOrganizationRole("id", RoleOwner))
	routeBuilder.ProtectedGETWithMetadata("/:id/invitations", "organization-list-invitations", handlers.ListInvitations, routeMetadata("List organization invitations"), p.MiddlewareRequireOrganizationRole("id", RoleOwner))
	routeBuilder.ProtectedPOSTWithMetadata("/invitations/accept", "organization-accept-invitation", handlers.AcceptInvitation, routeMetadata(
		"Accept organization invitation",
		limen.WithRouteAllowedContentTypes("application/json"),
	))
}

func (p *organizationPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	p.organization = newOrganizationSchema()
	p.membership = newMembershipSchema()
	p.invitation = newInvitationSchema()
	return []limen.SchemaIntrospector{
		buildOrganizationTableDef(schema, p.organization),
		buildMembershipTableDef(schema, p.membership),
		buildInvitationTableDef(schema, p.invitation),
	}
}

func (p *organizationPlugin) CreateOrganization(ctx context.Context, userID any, name string, slug string) (*Organization, error) {
	name = strings.TrimSpace(name)
	slug = strings.TrimSpace(slug)
	if name == "" {
		return nil, limen.NewLimenError("name is required", http.StatusUnprocessableEntity, nil)
	}
	if slug == "" {
		return nil, limen.NewLimenError("slug is required", http.StatusUnprocessableEntity, nil)
	}

	var created *Organization
	err := p.core.WithTransaction(ctx, func(txCtx context.Context) error {
		exists, err := p.core.Exists(txCtx, p.organization, []limen.Where{
			limen.Eq(p.organization.GetSlugField(), slug),
		})
		if err != nil {
			return err
		}
		if exists {
			return limen.NewLimenError("organization slug already exists", http.StatusConflict, nil)
		}

		now := time.Now()
		org := &Organization{Name: name, Slug: slug, CreatedAt: now, UpdatedAt: now}
		if err := p.core.Create(txCtx, p.organization, org, nil); err != nil {
			return err
		}
		stored, err := p.findOrganizationBySlug(txCtx, slug)
		if err != nil {
			return err
		}
		if _, err := p.addMember(txCtx, stored.ID, userID, p.defaultOwnerRole, true); err != nil {
			return err
		}
		created = stored
		return nil
	})
	return created, err
}

func (p *organizationPlugin) ListOrganizationsForUser(ctx context.Context, userID any) ([]*Organization, error) {
	memberships, err := p.findMembershipsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	orgs := make([]*Organization, 0, len(memberships))
	for _, membership := range memberships {
		org, err := p.findOrganizationByID(ctx, membership.OrganizationID)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

func (p *organizationPlugin) AddMember(ctx context.Context, organizationID any, userID any, role Role) (*Membership, error) {
	return p.addMember(ctx, organizationID, userID, role, false)
}

func (p *organizationPlugin) addMember(ctx context.Context, organizationID any, userID any, role Role, allowOwner bool) (*Membership, error) {
	role = normalizeRole(role)
	if role == "" || (role == RoleOwner && !allowOwner) {
		return nil, limen.NewLimenError("role is invalid", http.StatusUnprocessableEntity, nil)
	}
	exists, err := p.core.Exists(ctx, p.membership, []limen.Where{
		limen.Eq(p.membership.GetOrganizationIDField(), organizationID),
		limen.Eq(p.membership.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, limen.NewLimenError("organization member already exists", http.StatusConflict, nil)
	}

	now := time.Now()
	membership := &Membership{
		OrganizationID: organizationID,
		UserID:         userID,
		Role:           role,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := p.core.Create(ctx, p.membership, membership, nil); err != nil {
		return nil, err
	}
	return p.GetMembership(ctx, organizationID, userID)
}

func (p *organizationPlugin) RemoveMember(ctx context.Context, organizationID any, userID any) error {
	membership, err := p.GetMembership(ctx, organizationID, userID)
	if err != nil {
		return err
	}
	if membership.Role == RoleOwner {
		return limen.NewLimenError("owners cannot be removed by generic member removal", http.StatusForbidden, nil)
	}
	return p.core.Delete(ctx, p.membership, []limen.Where{
		limen.Eq(p.membership.GetOrganizationIDField(), organizationID),
		limen.Eq(p.membership.GetUserIDField(), userID),
	})
}

func (p *organizationPlugin) GetMembership(ctx context.Context, organizationID any, userID any) (*Membership, error) {
	model, err := p.core.FindOne(ctx, p.membership, []limen.Where{
		limen.Eq(p.membership.GetOrganizationIDField(), organizationID),
		limen.Eq(p.membership.GetUserIDField(), userID),
	}, nil)
	if err != nil {
		return nil, err
	}
	return model.(*Membership), nil
}

func (p *organizationPlugin) HasRole(ctx context.Context, organizationID any, userID any, roles ...Role) (bool, error) {
	membership, err := p.GetMembership(ctx, organizationID, userID)
	if err != nil {
		if err == limen.ErrRecordNotFound {
			return false, nil
		}
		return false, err
	}
	return roleAllowed(membership.Role, roles), nil
}

func (p *organizationPlugin) CreateInvitation(ctx context.Context, organizationID any, email string, role Role) (*Invitation, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, limen.NewLimenError("email is required", http.StatusUnprocessableEntity, nil)
	}
	role = normalizeRole(role)
	if role == "" || role == RoleOwner {
		return nil, limen.NewLimenError("role is invalid", http.StatusUnprocessableEntity, nil)
	}
	token, err := randomToken()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	invitation := &Invitation{
		OrganizationID: organizationID,
		Email:          email,
		Role:           role,
		tokenHash:      p.hashToken(token),
		ExpiresAt:      now.Add(p.invitationTTL),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := p.core.Create(ctx, p.invitation, invitation, nil); err != nil {
		return nil, err
	}
	stored, err := p.findInvitationByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	stored.Token = token
	return stored, nil
}

func (p *organizationPlugin) ListInvitations(ctx context.Context, organizationID any) ([]*Invitation, error) {
	models, err := p.core.FindMany(ctx, p.invitation, []limen.Where{
		limen.Eq(p.invitation.GetOrganizationIDField(), organizationID),
	})
	if err != nil {
		return nil, err
	}
	invitations := make([]*Invitation, 0, len(models))
	for _, model := range models {
		invitations = append(invitations, model.(*Invitation))
	}
	return invitations, nil
}

func (p *organizationPlugin) AcceptInvitation(ctx context.Context, userID any, token string) (*Membership, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, limen.NewLimenError("token is required", http.StatusUnprocessableEntity, nil)
	}
	var membership *Membership
	err := p.core.WithTransaction(ctx, func(txCtx context.Context) error {
		invitation, err := p.findInvitationByToken(txCtx, token)
		if err != nil {
			return err
		}
		if invitation.AcceptedAt != nil {
			return limen.NewLimenError("invitation already accepted", http.StatusConflict, nil)
		}
		if time.Now().After(invitation.ExpiresAt) {
			return limen.NewLimenError("invitation has expired", http.StatusGone, nil)
		}
		user, err := p.core.FindOne(txCtx, p.core.Schema.User, []limen.Where{
			limen.Eq(p.core.Schema.User.GetIDField(), userID),
		}, nil)
		if err != nil {
			return err
		}
		if !strings.EqualFold(user.(*limen.User).Email, invitation.Email) {
			return limen.NewLimenError("invitation email does not match user", http.StatusForbidden, nil)
		}

		exists, err := p.core.Exists(txCtx, p.membership, []limen.Where{
			limen.Eq(p.membership.GetOrganizationIDField(), invitation.OrganizationID),
			limen.Eq(p.membership.GetUserIDField(), userID),
		})
		if err != nil {
			return err
		}
		if exists {
			return limen.NewLimenError("organization member already exists", http.StatusConflict, nil)
		}

		now := time.Now()
		invitation.AcceptedAt = &now
		invitation.UpdatedAt = now
		affected, err := p.core.UpdateRawAffected(txCtx, p.invitation, invitation, []limen.Where{
			limen.Eq(p.invitation.GetIDField(), invitation.ID),
			limen.IsNull(p.invitation.GetAcceptedAtField()),
		}, true)
		if err != nil {
			return err
		}
		if affected != 1 {
			return limen.NewLimenError("invitation already accepted", http.StatusConflict, nil)
		}

		membership, err = p.addMember(txCtx, invitation.OrganizationID, userID, invitation.Role, false)
		return err
	})
	if err != nil {
		return nil, err
	}
	return membership, nil
}

func (p *organizationPlugin) MiddlewareRequireOrganizationRole(paramName string, roles ...Role) limen.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := limen.GetCurrentSessionFromCtx(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			organizationID := parseID(limen.GetParam(r, paramName))
			allowed, err := p.HasRole(r.Context(), organizationID, session.User.ID, roles...)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !allowed {
				http.Error(w, "organization access is forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (p *organizationPlugin) findOrganizationBySlug(ctx context.Context, slug string) (*Organization, error) {
	model, err := p.core.FindOne(ctx, p.organization, []limen.Where{
		limen.Eq(p.organization.GetSlugField(), slug),
	}, nil)
	if err != nil {
		return nil, err
	}
	return model.(*Organization), nil
}

func (p *organizationPlugin) findOrganizationByID(ctx context.Context, id any) (*Organization, error) {
	model, err := p.core.FindOne(ctx, p.organization, []limen.Where{
		limen.Eq(p.organization.GetIDField(), id),
	}, nil)
	if err != nil {
		return nil, err
	}
	return model.(*Organization), nil
}

func (p *organizationPlugin) findMembershipsByUser(ctx context.Context, userID any) ([]*Membership, error) {
	models, err := p.core.FindMany(ctx, p.membership, []limen.Where{
		limen.Eq(p.membership.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}
	memberships := make([]*Membership, 0, len(models))
	for _, model := range models {
		memberships = append(memberships, model.(*Membership))
	}
	return memberships, nil
}

func (p *organizationPlugin) findInvitationByToken(ctx context.Context, token string) (*Invitation, error) {
	model, err := p.core.FindOne(ctx, p.invitation, []limen.Where{
		limen.Eq(p.invitation.GetTokenField(), p.hashToken(token)),
	}, nil)
	if err != nil {
		return nil, err
	}
	return model.(*Invitation), nil
}

func (p *organizationPlugin) hashToken(token string) string {
	mac := hmac.New(sha256.New, p.core.Secret())
	_, _ = mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func normalizeRole(role Role) Role {
	switch role {
	case RoleOwner, RoleAdmin, RoleMember:
		return role
	default:
		return ""
	}
}

func roleAllowed(actual Role, required []Role) bool {
	if len(required) == 0 || actual == RoleOwner {
		return true
	}
	for _, role := range required {
		if actual == role {
			return true
		}
	}
	return false
}

func parseID(value string) any {
	if id, err := strconv.ParseInt(value, 10, 64); err == nil {
		return id
	}
	return value
}

func routeMetadata(summary string, opts ...limen.RouteMetadataOption) *limen.RouteMetadata {
	options := []limen.RouteMetadataOption{
		limen.WithRouteSummary(summary),
		limen.WithRouteTags("organizations"),
	}
	options = append(options, opts...)
	return limen.NewRouteMetadata(options...)
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
