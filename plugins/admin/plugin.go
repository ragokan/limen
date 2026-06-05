package admin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ragokan/limen"
)

const defaultBasePath = "/admin"

type adminPlugin struct {
	core   *limen.LimenCore
	config *config
}

type API interface {
	IsAdmin(user *limen.User) bool
	ListUsers(ctx context.Context) ([]map[string]any, error)
	GetUser(ctx context.Context, id any) (map[string]any, error)
	RevokeUserSessions(ctx context.Context, id any) error
	MiddlewareRequireAdmin() limen.Middleware
}

func New(opts ...ConfigOption) *adminPlugin {
	cfg := &config{
		adminEmails: make(map[string]struct{}),
		adminIDs:    make(map[string]struct{}),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &adminPlugin{config: cfg}
}

func Use(auth *limen.Limen) API {
	return limen.Use[API](auth, limen.PluginAdmin)
}

func (p *adminPlugin) Name() limen.PluginName {
	return limen.PluginAdmin
}

func (p *adminPlugin) Initialize(core *limen.LimenCore) error {
	if p.config == nil {
		return fmt.Errorf("admin: config is required")
	}
	p.core = core
	return nil
}

func (p *adminPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: defaultBasePath,
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRule("", 60, time.Minute),
		},
	}
}

func (p *adminPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	handlers := newHandlers(p, httpCore)
	adminOnly := p.MiddlewareRequireAdmin()
	routeBuilder.ProtectedGETWithMetadata("/users", "admin-list-users", handlers.ListUsers, routeMetadata("List users"), adminOnly)
	routeBuilder.ProtectedGETWithMetadata("/users/:id", "admin-get-user", handlers.GetUser, routeMetadata("Get user"), adminOnly)
	routeBuilder.ProtectedPOSTWithMetadata("/users/:id/revoke-sessions", "admin-revoke-user-sessions", handlers.RevokeUserSessions, routeMetadata(
		"Revoke user sessions",
		limen.WithRouteResponse(http.StatusNoContent, limen.OpenAPIResponse{Description: "No Content"}),
	), adminOnly)
}

func (p *adminPlugin) IsAdmin(user *limen.User) bool {
	if user == nil {
		return false
	}
	if len(p.config.adminEmails) == 0 && len(p.config.adminIDs) == 0 {
		return false
	}
	if _, ok := p.config.adminEmails[strings.ToLower(strings.TrimSpace(user.Email))]; ok && user.EmailVerifiedAt != nil {
		return true
	}
	_, ok := p.config.adminIDs[fmt.Sprint(user.ID)]
	return ok
}

func (p *adminPlugin) MiddlewareRequireAdmin() limen.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, err := limen.GetCurrentSessionFromCtx(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if len(p.config.adminEmails) == 0 && len(p.config.adminIDs) == 0 {
				http.Error(w, ErrAdminNotConfigured.Error(), ErrAdminNotConfigured.Status())
				return
			}
			if !p.IsAdmin(session.User) {
				http.Error(w, ErrAdminForbidden.Error(), ErrAdminForbidden.Status())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (p *adminPlugin) ListUsers(ctx context.Context) ([]map[string]any, error) {
	models, err := p.core.FindMany(ctx, p.core.Schema.User, nil)
	if err != nil {
		return nil, err
	}
	users := make([]map[string]any, 0, len(models))
	for _, model := range models {
		users = append(users, p.adminUserPayload(model.(*limen.User)))
	}
	return users, nil
}

func (p *adminPlugin) GetUser(ctx context.Context, id any) (map[string]any, error) {
	model, err := p.core.FindOne(ctx, p.core.Schema.User, []limen.Where{
		limen.Eq(p.core.Schema.User.GetIDField(), id),
	}, nil)
	if err != nil {
		return nil, err
	}
	return p.adminUserPayload(model.(*limen.User)), nil
}

func (p *adminPlugin) RevokeUserSessions(ctx context.Context, id any) error {
	return p.core.SessionManager.RevokeAllSessions(ctx, id)
}

func (p *adminPlugin) adminUserPayload(user *limen.User) map[string]any {
	raw := user.Raw()
	out := make(map[string]any, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	delete(out, p.core.Schema.User.GetPasswordField())
	return out
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
		limen.WithRouteTags("admin"),
	}
	options = append(options, opts...)
	return limen.NewRouteMetadata(options...)
}
