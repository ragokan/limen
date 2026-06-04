package oauthmicrosoft

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
				"oid":            "microsoft-user-1",
				"email":          "user@example.com",
				"email_verified": true,
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
	if info.ID != "microsoft-user-1" || info.Email != "user@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected user info: %#v", info)
	}
}

func TestGetUserInfo_EmailVerifiedRequiresTrustedClaim(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"oid":   "microsoft-user-1",
				"email": "user@example.com",
				"nonce": "nonce",
			}, nil
		}),
	)

	ctx := oauth.ContextWithIDTokenNonce(context.Background(), "nonce")
	info, err := provider.GetUserInfo(ctx, &oauth.TokenResponse{IDToken: "id-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.EmailVerified {
		t.Fatalf("expected email to be unverified without email_verified claim: %#v", info)
	}
}

func TestGetUserInfo_RejectsNonceMismatch(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithIDTokenVerifier(func(_ context.Context, _ string) (map[string]any, error) {
			return map[string]any{
				"oid":   "microsoft-user-1",
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

func TestMicrosoftEmailVerified_UsesVerifiedEmailArrays(t *testing.T) {
	t.Parallel()

	claims := map[string]any{
		"verified_primary_email": []any{"user@example.com"},
	}

	if !microsoftEmailVerified(claims, "user@example.com") {
		t.Fatal("expected verified primary email to mark the email verified")
	}
}

func TestMicrosoftTenantIssuerValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		tenant string
		issuer string
		want   bool
	}{
		{
			name:   "common accepts tenant GUID issuer",
			tenant: "common",
			issuer: "https://login.microsoftonline.com/72f988bf-86f1-41af-91ab-2d7cd011db47/v2.0",
			want:   true,
		},
		{
			name:   "organizations rejects non-GUID tenant",
			tenant: "organizations",
			issuer: "https://login.microsoftonline.com/evil/v2.0",
			want:   false,
		},
		{
			name:   "consumers accepts Microsoft consumer tenant",
			tenant: "consumers",
			issuer: "https://login.microsoftonline.com/9188040d-6c67-4c5b-b112-36a304b66dad/v2.0",
			want:   true,
		},
		{
			name:   "specific tenant requires exact issuer",
			tenant: "contoso.onmicrosoft.com",
			issuer: "https://login.microsoftonline.com/contoso.onmicrosoft.com/v2.0",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isMicrosoftTenantIssuer(defaultAuthority, tt.tenant, tt.issuer)
			if got != tt.want {
				t.Fatalf("isMicrosoftTenantIssuer() = %v, want %v", got, tt.want)
			}
		})
	}
}
