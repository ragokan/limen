// Package oauthgoogle provides a Google OAuth provider for the Limen OAuth plugin.
package oauthgoogle

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/oauth2"

	"github.com/thecodearcher/limen/plugins/oauth"
)

var googleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
	TokenURL: "https://oauth2.googleapis.com/token",
}

// New creates a Google OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		clientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		clientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		scopes:       []string{"openid", "email", "profile"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newGoogleProvider(cfg)
}

type googleProvider struct {
	oauthConfig *oauth2.Config
	config      *config
}

func newGoogleProvider(cfg *config) *googleProvider {
	scopes := cfg.scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}
	if cfg.verifyIDToken == nil {
		cfg.verifyIDToken = oauth.NewIDTokenVerifier("https://accounts.google.com", cfg.clientID)
	}
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       scopes,
		Endpoint:     googleEndpoint,
	}
	return &googleProvider{oauthConfig: config, config: cfg}
}

func (g *googleProvider) Name() string {
	return "google"
}

func (g *googleProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	var authOpts []oauth2.AuthCodeOption

	for key, value := range g.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return g.oauthConfig, authOpts
}

func (g *googleProvider) IDTokenNonceEnabled() bool {
	return true
}

func (g *googleProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	if token.IDToken == "" {
		return nil, errors.New("google: id_token required; include openid scope")
	}
	claims, err := g.config.verifyIDToken(ctx, token.IDToken)
	if err != nil {
		return nil, fmt.Errorf("google: %w", err)
	}
	if err := oauth.VerifyIDTokenNonce(claims, oauth.IDTokenNonce(ctx)); err != nil {
		return nil, fmt.Errorf("google: %w", err)
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return nil, errors.New("google: id token missing sub claim")
	}
	email, _ := claims["email"].(string)
	if email == "" {
		return nil, errors.New("google: id token missing email claim")
	}
	emailVerified, _ := claims["email_verified"].(bool)
	name, _ := claims["name"].(string)
	picture, _ := claims["picture"].(string)

	return &oauth.ProviderUserInfo{
		ID:            sub,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		AvatarURL:     picture,
		Raw:           claims,
	}, nil
}
