package oauthapple

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
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
				"sub":            "apple-user-1",
				"email":          "user@example.com",
				"email_verified": "true",
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
	if info.ID != "apple-user-1" || info.Email != "user@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected user info: %#v", info)
	}
}

func TestOAuth2Config_UsesAppleClientSecretPost(t *testing.T) {
	t.Parallel()

	provider := New(WithScopes())
	cfg, _ := provider.OAuth2Config()
	if cfg.Endpoint.AuthURL != "https://appleid.apple.com/auth/authorize" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://appleid.apple.com/auth/token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
	if cfg.Endpoint.AuthStyle != oauth2.AuthStyleInParams {
		t.Fatalf("AuthStyle = %v", cfg.Endpoint.AuthStyle)
	}
	assertScopes(t, cfg.Scopes, "name", "email")
	rm, ok := provider.(oauth.ResponseModeProvider)
	if !ok || rm.ResponseMode() != oauth.ResponseModeFormPost {
		t.Fatal("ResponseMode must be form_post")
	}
}

func TestGetUserInfo_AcceptsSHA256NonceClaim(t *testing.T) {
	t.Parallel()

	nonce := "nonce-value"
	sum := sha256.Sum256([]byte(nonce))
	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"sub":            "apple-user-1",
				"email":          "user@example.com",
				"email_verified": true,
				"nonce":          hex.EncodeToString(sum[:]),
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), nonce)
	info, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if !info.EmailVerified {
		t.Fatalf("expected verified email: %#v", info)
	}
}

func TestExtractNameFromParams(t *testing.T) {
	t.Parallel()

	name := extractNameFromParams(mapValues("user", `{"name":{"firstName":"Test","lastName":"User"}}`))
	if name != "Test User" {
		t.Fatalf("name = %q", name)
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
				"email_verified": "true",
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
				"sub":   "apple-user-1",
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

func TestGetUserInfo_MapsFalseEmailVerifiedClaims(t *testing.T) {
	t.Parallel()

	for _, raw := range []any{"false", false} {
		provider := New(
			WithClientID("client-id"),
			WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
				return map[string]any{
					"sub":            "apple-user-1",
					"email":          "user@example.com",
					"email_verified": raw,
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
			t.Fatalf("expected unverified email for %v: %#v", raw, info)
		}
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

func mapValues(key, value string) url.Values {
	return url.Values{key: []string{value}}
}
