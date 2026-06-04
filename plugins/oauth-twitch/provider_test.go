package oauthtwitch

import (
	"context"
	"testing"

	"golang.org/x/oauth2"

	"github.com/ragokan/limen/plugins/oauth"
)

func TestGetUserInfo_UsesIDTokenVerifier(t *testing.T) {
	t.Parallel()

	called := false
	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, idToken string) (map[string]any, error) {
			called = true
			if idToken != "id-token" {
				t.Fatalf("unexpected id token: %s", idToken)
			}
			return map[string]any{
				"sub":                "twitch-user-1",
				"email":              "user@example.com",
				"email_verified":     true,
				"preferred_username": "testuser",
				"nonce":              "nonce",
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce")
	info, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if !called {
		t.Fatal("expected verifier to be called")
	}
	if info.ID != "twitch-user-1" || info.Email != "user@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected user info: %#v", info)
	}
}

func TestTwitchEndpointAuthStyle(t *testing.T) {
	t.Parallel()

	provider := New().(*twitchProvider)
	cfg, _ := provider.OAuth2Config()

	if cfg.Endpoint.AuthStyle != oauth2.AuthStyleInParams {
		t.Fatalf("AuthStyle = %v, want AuthStyleInParams", cfg.Endpoint.AuthStyle)
	}
}

func TestGetUserInfo_RejectsNonceMismatch(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"sub":   "twitch-user-1",
				"email": "user@example.com",
				"nonce": "other",
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce")
	_, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err == nil {
		t.Fatal("expected nonce mismatch")
	}
}
