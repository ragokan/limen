// Package oauthtwitch provides a Twitch OAuth 2.0 / OpenID Connect provider for the Limen OAuth plugin.
package oauthtwitch

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/oauth2"

	"github.com/ragokan/limen/plugins/oauth"
)

var twitchEndpoint = oauth2.Endpoint{ // #nosec G101 -- Twitch OAuth endpoint URLs are public provider metadata, not credentials.
	AuthURL:   "https://id.twitch.tv/oauth2/authorize",
	TokenURL:  "https://id.twitch.tv/oauth2/token",
	AuthStyle: oauth2.AuthStyleInParams,
}

const claimsParam = `{"id_token":{"email":null,"email_verified":null,"picture":null,"preferred_username":null}}`

// New creates a Twitch OAuth provider that implements oauth.Provider.
func New(opts ...ConfigOption) oauth.Provider {
	cfg := &config{
		clientID:     os.Getenv("TWITCH_CLIENT_ID"),
		clientSecret: os.Getenv("TWITCH_CLIENT_SECRET"),
		scopes:       []string{"openid", "user:read:email"},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return newTwitchProvider(cfg)
}

type twitchProvider struct {
	oauthConfig *oauth2.Config
	config      *config
}

func newTwitchProvider(cfg *config) *twitchProvider {
	if cfg.verifyIDToken == nil {
		cfg.verifyIDToken = oauth.NewIDTokenVerifier("https://id.twitch.tv/oauth2", cfg.clientID)
	}
	config := &oauth2.Config{
		ClientID:     cfg.clientID,
		ClientSecret: cfg.clientSecret,
		RedirectURL:  cfg.redirectURL,
		Scopes:       cfg.scopes,
		Endpoint:     twitchEndpoint,
	}
	return &twitchProvider{oauthConfig: config, config: cfg}
}

func (t *twitchProvider) Name() string {
	return "twitch"
}

func (t *twitchProvider) OAuth2Config() (*oauth2.Config, []oauth2.AuthCodeOption) {
	authOpts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("claims", claimsParam),
	}
	for key, value := range t.config.options {
		authOpts = append(authOpts, oauth2.SetAuthURLParam(key, value))
	}
	return t.oauthConfig, authOpts
}

func (t *twitchProvider) IDTokenNonceEnabled() bool {
	return true
}

func (t *twitchProvider) GetUserInfo(ctx context.Context, token *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
	if token.IDToken == "" {
		return nil, errors.New("twitch: id_token required; include openid scope")
	}
	claims, err := t.config.verifyIDToken(ctx, token.IDToken)
	if err != nil {
		return nil, fmt.Errorf("twitch: %w", err)
	}
	if err := oauth.VerifyIDTokenNonce(claims, oauth.IDTokenNonce(ctx)); err != nil {
		return nil, fmt.Errorf("twitch: %w", err)
	}

	id, _ := claims["sub"].(string)
	if id == "" {
		return nil, errors.New("twitch: missing sub claim")
	}

	email, _ := claims["email"].(string)
	if email == "" {
		return nil, errors.New("twitch: missing email claim; include user:read:email scope and claims")
	}

	name, _ := claims["preferred_username"].(string)
	picture, _ := claims["picture"].(string)

	emailVerified := false
	if v, ok := claims["email_verified"]; ok {
		switch b := v.(type) {
		case bool:
			emailVerified = b
		case string:
			emailVerified = b == "true" || b == "1"
		}
	}

	return &oauth.ProviderUserInfo{
		ID:            id,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		AvatarURL:     picture,
		Raw:           claims,
	}, nil
}
