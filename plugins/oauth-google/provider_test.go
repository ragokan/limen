package oauthgoogle

import (
	"context"
	"testing"

	"github.com/thecodearcher/limen/plugins/oauth"
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
				"sub":            "google-user-1",
				"email":          "user@example.com",
				"email_verified": true,
				"name":           "Test User",
				"picture":        "https://example.com/avatar.png",
				"nonce":          "nonce-value",
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce-value")
	info, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if !called {
		t.Fatal("expected verifier to be called")
	}
	if info.ID != "google-user-1" || info.Email != "user@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected user info: %#v", info)
	}
}

func TestOAuth2Config_UsesCurrentGoogleOIDCEndpoints(t *testing.T) {
	t.Parallel()

	provider := New()
	cfg, _ := provider.OAuth2Config()
	if cfg.Endpoint.AuthURL != "https://accounts.google.com/o/oauth2/v2/auth" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://oauth2.googleapis.com/token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
}

func TestGetUserInfo_RejectsNonceMismatch(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"sub":            "google-user-1",
				"email":          "user@example.com",
				"email_verified": true,
				"nonce":          "other-nonce",
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce-value")
	_, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err == nil {
		t.Fatal("expected nonce mismatch error")
	}
}
