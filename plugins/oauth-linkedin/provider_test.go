package oauthlinkedin

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ragokan/limen/plugins/oauth"
)

func TestLinkedInIssuer(t *testing.T) {
	t.Parallel()

	if linkedinIssuer != "https://www.linkedin.com/oauth" {
		t.Fatalf("linkedinIssuer = %q", linkedinIssuer)
	}
}

func TestOAuth2Config_UsesLinkedInOIDCEndpointsAndDefaults(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithRedirectURL("https://app.example/callback"),
		WithScopes(),
		WithOption("prompt", "consent"),
	)
	cfg, opts := provider.OAuth2Config()
	if cfg.Endpoint.AuthURL != "https://www.linkedin.com/oauth/v2/authorization" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://www.linkedin.com/oauth/v2/accessToken" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
	assertScopes(t, cfg.Scopes, "openid", "profile", "email")
	if pkce, ok := provider.(oauth.PKCEEnabledProvider); !ok || pkce.PKCEEnabled() {
		t.Fatal("LinkedIn web auth-code flow must disable PKCE")
	}
	authURL := cfg.AuthCodeURL("state", opts...)
	if !strings.Contains(authURL, "prompt=consent") {
		t.Fatalf("auth URL missing option: %s", authURL)
	}
}

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
				"sub":            "linkedin-user-1",
				"email":          "user@example.com",
				"email_verified": "true",
				"name":           "Test User",
				"nonce":          "nonce",
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
	if info.ID != "linkedin-user-1" || info.Email != "user@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected user info: %#v", info)
	}
}

func TestGetUserInfo_MapsBooleanEmailVerifiedClaim(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"sub":            "linkedin-user-1",
				"email":          "user@example.com",
				"email_verified": true,
				"nonce":          "nonce",
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce")
	info, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if !info.EmailVerified {
		t.Fatalf("expected verified email: %#v", info)
	}
}

func TestGetUserInfo_RejectsNonceMismatch(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"sub":   "linkedin-user-1",
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
				"sub":   "linkedin-user-1",
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

func TestGetUserInfo_PropagatesVerifierError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("verifier failed")
	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return nil, sentinel
		}),
	)

	_, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{IDToken: "id-token"})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
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
