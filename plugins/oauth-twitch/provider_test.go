package oauthtwitch

import (
	"context"
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
				"sub":                "twitch-user-1",
				"email":              "user@example.com",
				"email_verified":     true,
				"preferred_username": "testuser",
			}, nil
		}),
	)

	info, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{IDToken: "id-token"})
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
