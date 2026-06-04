package oauthlinkedin

import (
	"context"
	"testing"

	"github.com/ragokan/limen/plugins/oauth"
)

func TestLinkedInIssuer(t *testing.T) {
	t.Parallel()

	if linkedinIssuer != "https://www.linkedin.com/oauth" {
		t.Fatalf("linkedinIssuer = %q", linkedinIssuer)
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
