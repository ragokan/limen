package oauthgeneric

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/ragokan/limen/plugins/oauth"
)

// New creates a generic OAuth provider that implements oauth.Provider.
// Panics if required options are missing: name, clientID, clientSecret, authorizationURL, tokenURL.
// For user info, provide one of: WithGetUserInfo (full custom), WithUserInfoURL + WithMapUserInfo,
// or WithMapUserInfo alone (will use the id_token claims when the provider returns one).
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	cfg.resolveDiscovery()
	cfg.validate()
	cfg.resolveDefaults()
	if cfg.verifyIDToken == nil && cfg.issuer != "" {
		cfg.verifyIDToken = oauth.NewIDTokenVerifier(cfg.issuer, cfg.clientID)
	}

	return &genericProvider{
		config: cfg,
		scopes: cfg.scopes,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type genericProvider struct {
	config *config
	scopes []string
	client *http.Client
}

func (g *genericProvider) Name() string {
	return g.config.name
}

func (g *genericProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	endpoint := oauth2.Endpoint{
		AuthURL:  g.config.authorizationURL,
		TokenURL: g.config.tokenURL,
	}
	cfg := &oauth2.Config{
		ClientID:     g.config.clientID,
		ClientSecret: g.config.clientSecret,
		RedirectURL:  g.config.redirectURL,
		Scopes:       g.scopes,
		Endpoint:     endpoint,
	}

	var authOpts []oauth2.AuthCodeOption
	for key, value := range g.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return cfg, authOpts
}

func (g *genericProvider) IDTokenNonceEnabled() bool {
	return g.config.verifyIDToken != nil || g.config.issuer != ""
}

func (g *genericProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	if g.config.getUserInfo != nil {
		return g.config.getUserInfo(ctx, token)
	}
	if g.config.userInfoURL != "" {
		info, err := g.fetchUserInfoFromURL(ctx, token)
		if err != nil {
			return nil, err
		}
		if token.IDToken != "" && g.config.verifyIDToken != nil {
			claims, err := g.verifiedIDTokenClaims(ctx, token.IDToken)
			if err != nil {
				return nil, err
			}
			sub, _ := claims["sub"].(string)
			if sub != "" && info.ID != "" && sub != info.ID {
				return nil, fmt.Errorf("userinfo subject does not match id_token subject")
			}
		}
		return info, nil
	}
	return g.userInfoFromIDToken(ctx, token.IDToken)
}

func (g *genericProvider) userInfoFromIDToken(ctx context.Context, idToken string) (*oauth.ProviderUserInfo, error) {
	if idToken == "" {
		return nil, fmt.Errorf("id_token is required when no userinfo endpoint is configured")
	}

	verifier := g.config.verifyIDToken
	if verifier == nil {
		if g.config.issuer == "" {
			return nil, fmt.Errorf("issuer is required to verify id_token claims")
		}
		return nil, fmt.Errorf("id_token verifier is not configured")
	}

	claims, err := g.verifiedIDTokenClaims(ctx, idToken)
	if err != nil {
		return nil, err
	}
	info, err := g.config.mapUserInfo(claims)
	if err != nil {
		return nil, err
	}
	info.Raw = claims
	return info, nil
}

func (g *genericProvider) verifiedIDTokenClaims(ctx context.Context, idToken string) (map[string]any, error) {
	verifier := g.config.verifyIDToken
	if verifier == nil {
		return nil, fmt.Errorf("id_token verifier is not configured")
	}
	claims, err := verifier(ctx, idToken)
	if err != nil {
		return nil, err
	}
	if g.IDTokenNonceEnabled() {
		if err := oauth.VerifyIDTokenNonce(claims, oauth.IDTokenNonce(ctx)); err != nil {
			return nil, err
		}
	}
	return claims, nil
}

func (g *genericProvider) fetchUserInfoFromURL(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.config.userInfoURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo request failed: %s", resp.Status)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	info, err := g.config.mapUserInfo(raw)
	if err != nil {
		return nil, err
	}
	info.Raw = raw
	return info, nil
}

func (g *genericProvider) ExchangeAuthorizationCode(ctx context.Context, code, codeVerifier, redirectURI string) (*oauth.TokenResponse, error) {
	if g.config.exchangeTokens != nil {
		return g.config.exchangeTokens(ctx, code, codeVerifier, redirectURI)
	}
	cfg, _ := g.OAuth2Config()
	cfg.RedirectURL = redirectURI
	return oauth.ExchangeCode(ctx, cfg, code, codeVerifier)
}

// RefreshToken implements oauth.TokenRefresher. When WithRefreshTokens is set, the custom
// function is used; otherwise the standard oauth2 token refresh flow is used.
func (g *genericProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth.TokenResponse, error) {
	if g.config.refreshTokens != nil {
		return g.config.refreshTokens(ctx, refreshToken)
	}
	cfg, _ := g.OAuth2Config()
	return oauth.RefreshToken(ctx, cfg, refreshToken)
}
