// Package oauthmicrosoft provides a Microsoft OAuth provider for the Limen OAuth plugin.
package oauthmicrosoft

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/ragokan/limen/plugins/oauth"
)

const (
	defaultTenant       = "common"
	organizationsTenant = "organizations"
	consumersTenant     = "consumers"
	defaultAuthority    = "https://login.microsoftonline.com"
	emailClaim          = "email"
)

func microsoftEndpoint(authority string) oauth2.Endpoint {
	return oauth2.Endpoint{
		AuthURL:  authority + "/oauth2/v2.0/authorize",
		TokenURL: authority + "/oauth2/v2.0/token",
	}
}

// New creates a Microsoft OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		clientID:     os.Getenv("MICROSOFT_CLIENT_ID"),
		clientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET"),
		scopes:       []string{"openid", "profile", emailClaim},
		tenant:       defaultTenant,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newMicrosoftProvider(cfg)
}

type microsoftProvider struct {
	oauthConfig *oauth2.Config
	config      *config
}

func newMicrosoftProvider(cfg *config) *microsoftProvider {
	tenant := cfg.tenant
	if tenant == "" {
		tenant = defaultTenant
	}

	authority := strings.TrimRight(cfg.authorityURL, "/")
	if authority == "" {
		authority = defaultAuthority + "/" + tenant
	}
	issuer := authority + "/v2.0"
	if cfg.verifyIDToken == nil {
		if cfg.authorityURL == "" && isMicrosoftSharedTenant(tenant) {
			cfg.verifyIDToken = newMicrosoftSharedTenantIDTokenVerifier(defaultAuthority, tenant, cfg.clientID)
		} else {
			cfg.verifyIDToken = oauth.NewIDTokenVerifier(issuer, cfg.clientID)
		}
	}
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       cfg.scopes,
		Endpoint:     microsoftEndpoint(authority),
	}
	return &microsoftProvider{oauthConfig: config, config: cfg}
}

func isMicrosoftSharedTenant(tenant string) bool {
	return tenant == defaultTenant || tenant == organizationsTenant || tenant == consumersTenant
}

func newMicrosoftSharedTenantIDTokenVerifier(authorityBase, tenant, clientID string) oauth.IDTokenVerifier {
	discoveryIssuer := strings.TrimRight(authorityBase, "/") + "/" + tenant + "/v2.0"
	expectedDiscoveryIssuer := strings.TrimRight(authorityBase, "/") + "/{tenantid}/v2.0"
	if tenant == consumersTenant {
		expectedDiscoveryIssuer = strings.TrimRight(authorityBase, "/") + "/9188040d-6c67-4c5b-b112-36a304b66dad/v2.0"
	}

	var mu sync.Mutex
	var verifier *oidc.IDTokenVerifier

	return func(ctx context.Context, idToken string) (map[string]any, error) {
		mu.Lock()
		if verifier == nil {
			providerCtx := oidc.InsecureIssuerURLContext(ctx, expectedDiscoveryIssuer)
			provider, err := oidc.NewProvider(providerCtx, discoveryIssuer)
			if err != nil {
				mu.Unlock()
				return nil, fmt.Errorf("microsoft: id token provider discovery failed: %w", err)
			}
			verifier = provider.Verifier(&oidc.Config{
				ClientID:        clientID,
				SkipIssuerCheck: true,
			})
		}
		current := verifier
		mu.Unlock()

		verified, err := current.Verify(ctx, idToken)
		if err != nil {
			return nil, fmt.Errorf("microsoft: id token verification failed: %w", err)
		}
		if !isMicrosoftTenantIssuer(authorityBase, tenant, verified.Issuer) {
			return nil, fmt.Errorf("microsoft: id token issuer is not trusted")
		}

		var claims map[string]any
		if err := verified.Claims(&claims); err != nil {
			return nil, fmt.Errorf("microsoft: id token claims decode failed: %w", err)
		}
		return claims, nil
	}
}

func isMicrosoftTenantIssuer(authorityBase, tenant, issuer string) bool {
	base := strings.TrimRight(authorityBase, "/")
	if tenant == consumersTenant {
		return issuer == base+"/9188040d-6c67-4c5b-b112-36a304b66dad/v2.0"
	}
	if tenant != defaultTenant && tenant != organizationsTenant {
		return issuer == base+"/"+tenant+"/v2.0"
	}

	prefix := base + "/"
	suffix := "/v2.0"
	if !strings.HasPrefix(issuer, prefix) || !strings.HasSuffix(issuer, suffix) {
		return false
	}
	tenantID := strings.TrimSuffix(strings.TrimPrefix(issuer, prefix), suffix)
	return isGUID(tenantID)
}

func isGUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, ch := range value {
		switch i {
		case 8, 13, 18, 23:
			if ch != '-' {
				return false
			}
		default:
			if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') && (ch < 'A' || ch > 'F') {
				return false
			}
		}
	}
	return true
}

func (m *microsoftProvider) Name() string {
	return "microsoft"
}

func (m *microsoftProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption
	for key, value := range m.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return m.oauthConfig, authOpts
}

func (m *microsoftProvider) IDTokenNonceEnabled() bool {
	return true
}

func (m *microsoftProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	if token.IDToken == "" {
		return nil, errors.New("microsoft: id_token required; include openid scope")
	}
	claims, err := m.config.verifyIDToken(ctx, token.IDToken)
	if err != nil {
		return nil, fmt.Errorf("microsoft: %w", err)
	}
	if err := oauth.VerifyIDTokenNonce(claims, oauth.IDTokenNonce(ctx)); err != nil {
		return nil, fmt.Errorf("microsoft: %w", err)
	}

	oid, _ := claims["oid"].(string)
	if oid == "" {
		return nil, errors.New("microsoft: id token missing oid claim")
	}

	email := extractEmail(claims)
	name, _ := claims["name"].(string)
	emailVerified := microsoftEmailVerified(claims, email)

	return &oauth.ProviderUserInfo{
		ID:            oid,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		Raw:           claims,
	}, nil
}

// extractEmail returns the user's email from the ID token claims.
// "email" is preferred; falls back to "preferred_username" which Microsoft
// typically populates with the user's UPN or email address.
func extractEmail(claims map[string]any) string {
	if email, _ := claims[emailClaim].(string); email != "" {
		return email
	}
	upn, _ := claims["preferred_username"].(string)
	return upn
}

func microsoftEmailVerified(claims map[string]any, email string) bool {
	if verified, ok := claims["email_verified"].(bool); ok {
		return verified
	}
	if verified, ok := claims["email_verified"].(string); ok {
		return verified == "true" || verified == "1"
	}
	if email == "" {
		return false
	}
	return stringSliceClaimContains(claims["verified_primary_email"], email) ||
		stringSliceClaimContains(claims["verified_secondary_email"], email)
}

func stringSliceClaimContains(value any, needle string) bool {
	switch values := value.(type) {
	case []string:
		for _, value := range values {
			if value == needle {
				return true
			}
		}
	case []any:
		for _, value := range values {
			if value == needle {
				return true
			}
		}
	}
	return false
}
