package oauthgeneric

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ragokan/limen/plugins/oauth"
)

func TestGetUserInfo_MapsVerifiedIDTokenClaims(t *testing.T) {
	t.Parallel()

	called := false
	provider := New(
		WithName("generic"),
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithAuthorizationURL("https://provider.example.com/oauth/authorize"),
		WithTokenURL("https://provider.example.com/oauth/token"),
		WithIssuer("https://provider.example.com"),
		WithIDTokenVerifier(func(_ context.Context, idToken string) (map[string]any, error) {
			called = true
			if idToken != "id-token" {
				t.Fatalf("unexpected id token: %s", idToken)
			}
			return map[string]any{
				"sub":            "generic-user-1",
				"email":          "user@example.com",
				"email_verified": true,
				"nonce":          "nonce",
			}, nil
		}),
		WithMapUserInfo(func(raw map[string]any) (*oauth.ProviderUserInfo, error) {
			id, _ := raw["sub"].(string)
			email, _ := raw["email"].(string)
			emailVerified, _ := raw["email_verified"].(bool)
			return &oauth.ProviderUserInfo{
				ID:            id,
				Email:         email,
				EmailVerified: emailVerified,
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
	if info.ID != "generic-user-1" || info.Email != "user@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected user info: %#v", info)
	}
	if info.Raw == nil {
		t.Fatal("expected raw claims to be set")
	}
}

func TestGetUserInfo_RejectsMismatchedUserInfoSubject(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"sub":            "userinfo-user",
			"email":          "user@example.com",
			"email_verified": true,
		}); err != nil {
			t.Fatalf("write userinfo: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	provider := New(
		WithName("generic"),
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithAuthorizationURL("https://provider.example.com/oauth/authorize"),
		WithTokenURL("https://provider.example.com/oauth/token"),
		WithUserInfoURL(server.URL),
		WithIDTokenVerifier(func(_ context.Context, idToken string) (map[string]any, error) {
			if idToken != "id-token" {
				t.Fatalf("unexpected id token: %s", idToken)
			}
			return map[string]any{
				"sub":   "id-token-user",
				"nonce": "nonce",
			}, nil
		}),
		WithMapUserInfo(func(raw map[string]any) (*oauth.ProviderUserInfo, error) {
			id, _ := raw["sub"].(string)
			email, _ := raw["email"].(string)
			emailVerified, _ := raw["email_verified"].(bool)
			return &oauth.ProviderUserInfo{
				ID:            id,
				Email:         email,
				EmailVerified: emailVerified,
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce")
	_, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{
		AccessToken: "access-token",
		IDToken:     "id-token",
	})
	if err == nil {
		t.Fatal("expected mismatched subject to be rejected")
	}
}

func TestGetUserInfo_RejectsIDTokenNonceMismatch(t *testing.T) {
	t.Parallel()

	provider := New(
		WithName("generic"),
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithAuthorizationURL("https://provider.example.com/oauth/authorize"),
		WithTokenURL("https://provider.example.com/oauth/token"),
		WithIssuer("https://provider.example.com"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"sub":   "generic-user-1",
				"email": "user@example.com",
				"nonce": "other",
			}, nil
		}),
		WithMapUserInfo(func(raw map[string]any) (*oauth.ProviderUserInfo, error) {
			id, _ := raw["sub"].(string)
			email, _ := raw["email"].(string)
			return &oauth.ProviderUserInfo{ID: id, Email: email}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce")
	_, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err == nil {
		t.Fatal("expected nonce mismatch")
	}
}
