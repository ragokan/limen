package apikey

import (
	"context"
	"fmt"
	"time"

	"github.com/ragokan/limen"
)

type apiKeyPlugin struct {
	core   *limen.LimenCore
	config *config
	schema *apiKeySchema
}

func New(opts ...ConfigOption) *apiKeyPlugin {
	cfg := &config{
		headerName:       defaultHeaderName,
		authorization:    defaultAuthorization,
		keyPrefix:        defaultKeyPrefix,
		storedPrefixSize: defaultStoredPrefixSize,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &apiKeyPlugin{config: cfg}
}

func (p *apiKeyPlugin) Name() limen.PluginName {
	return limen.PluginAPIKey
}

func (p *apiKeyPlugin) Initialize(core *limen.LimenCore) error {
	if p.config == nil {
		return fmt.Errorf("api-key: config is required")
	}
	if p.config.headerName == "" {
		return fmt.Errorf("api-key: header name is required")
	}
	if p.config.authorization == "" {
		return fmt.Errorf("api-key: authorization scheme prefix is required")
	}
	if p.config.keyPrefix == "" {
		return fmt.Errorf("api-key: key prefix is required")
	}
	if p.config.storedPrefixSize <= len(p.config.keyPrefix) {
		return fmt.Errorf("api-key: stored prefix size must be greater than key prefix length")
	}
	if len(core.Secret()) != 32 {
		return fmt.Errorf("api-key: Limen secret must be 32 bytes")
	}

	p.core = core
	return nil
}

func (p *apiKeyPlugin) PluginHTTPConfig() limen.PluginHTTPConfig {
	return limen.PluginHTTPConfig{
		BasePath: defaultBasePath,
		RateLimitRules: []*limen.RateLimitRule{
			limen.NewRateLimitRule("", 20, time.Minute),
		},
	}
}

func (p *apiKeyPlugin) RegisterRoutes(httpCore *limen.LimenHTTPCore, routeBuilder *limen.RouteBuilder) {
	handlers := newHandlers(p, httpCore)
	routeBuilder.ProtectedPOSTWithMetadata("", "api-key-create", handlers.Create, routeMetadata(
		"Create API key",
		limen.WithRouteAllowedContentTypes("application/json"),
		limen.WithRouteResponse(201, limen.OpenAPIResponse{Description: "Created"}),
	))
	routeBuilder.ProtectedGETWithMetadata("", "api-key-list", handlers.List, routeMetadata("List API keys"))
	routeBuilder.ProtectedDELETEWithMetadata("/:id", "api-key-revoke", handlers.Revoke, routeMetadata(
		"Revoke API key",
		limen.WithRouteResponse(204, limen.OpenAPIResponse{Description: "No Content"}),
	))
}

func (p *apiKeyPlugin) GetSchemas(schema *limen.SchemaConfig) []limen.SchemaIntrospector {
	p.schema = newAPIKeySchema()
	return []limen.SchemaIntrospector{
		buildAPIKeyTableDef(schema, p.schema),
	}
}

func routeMetadata(summary string, opts ...limen.RouteMetadataOption) *limen.RouteMetadata {
	options := []limen.RouteMetadataOption{
		limen.WithRouteSummary(summary),
		limen.WithRouteTags("api-keys"),
	}
	options = append(options, opts...)
	return limen.NewRouteMetadata(options...)
}

type API interface {
	CreateAPIKey(ctx context.Context, userID any, name string, opts ...CreateAPIKeyOption) (*CreatedAPIKey, error)
	ListAPIKeys(ctx context.Context, userID any) ([]*APIKey, error)
	RevokeAPIKey(ctx context.Context, userID any, id any) error
	ValidateAPIKey(ctx context.Context, key string, scopes ...string) (*APIKey, error)
	MiddlewareRequireAPIKey(scopes ...string) limen.Middleware
}

func Use(auth *limen.Limen) API {
	return limen.Use[API](auth, limen.PluginAPIKey)
}
