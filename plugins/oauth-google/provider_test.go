package oauthgoogle

import (
	"context"
	"strings"
	"testing"

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

	provider := New(
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithRedirectURL("https://app.example/callback"),
		WithOption("prompt", "consent"),
	)
	cfg, _ := provider.OAuth2Config()
	if cfg.Endpoint.AuthURL != "https://accounts.google.com/o/oauth2/v2/auth" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://oauth2.googleapis.com/token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
	if cfg.ClientID != "client-id" || cfg.ClientSecret != "client-secret" || cfg.RedirectURL != "https://app.example/callback" {
		t.Fatalf("unexpected OAuth2 config: %#v", cfg)
	}
	assertScopes(t, cfg.Scopes, "openid", "email", "profile")
	_, opts := provider.OAuth2Config()
	if len(opts) != 1 {
		t.Fatalf("auth opts len = %d, want 1", len(opts))
	}
	authURL := cfg.AuthCodeURL("state", opts...)
	if !strings.Contains(authURL, "prompt=consent") {
		t.Fatalf("auth URL missing prompt option: %s", authURL)
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

func TestGetUserInfo_RequiresIDToken(t *testing.T) {
	t.Parallel()

	_, err := New(WithClientID("client-id")).GetUserInfo(context.Background(), &oauth.TokenResponse{})
	if err == nil {
		t.Fatal("expected missing id_token error")
	}
}

func TestGetUserInfo_RejectsMissingSub(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"email":          "user@example.com",
				"email_verified": true,
				"nonce":          "nonce",
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce")
	_, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err == nil {
		t.Fatal("expected missing sub error")
	}
}

func TestGetUserInfo_RejectsMissingEmail(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"sub":   "google-user-1",
				"nonce": "nonce",
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce")
	_, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err == nil {
		t.Fatal("expected missing email error")
	}
}

func TestGetUserInfo_MapsUnverifiedEmail(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"sub":            "google-user-1",
				"email":          "user@example.com",
				"email_verified": false,
				"nonce":          "nonce",
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce")
	info, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.EmailVerified {
		t.Fatalf("expected unverified email: %#v", info)
	}
}

func assertScopes(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Scopes = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Scopes = %#v, want %#v", got, want)
		}
	}
}
