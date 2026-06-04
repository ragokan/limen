package oauthapple

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"testing"

	"golang.org/x/oauth2"

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

	provider := New()
	cfg, _ := provider.OAuth2Config()
	if cfg.Endpoint.AuthStyle != oauth2.AuthStyleInParams {
		t.Fatalf("AuthStyle = %v", cfg.Endpoint.AuthStyle)
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

func mapValues(key, value string) url.Values {
	return url.Values{key: []string{value}}
}
