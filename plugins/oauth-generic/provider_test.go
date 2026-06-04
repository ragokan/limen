package oauthgeneric

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestDiscoveryPopulatesEndpointsAndIssuer(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept = %q", got)
		}
		writeJSON(t, w, map[string]any{
			"issuer":                 "https://issuer.example.com",
			"authorization_endpoint": "https://issuer.example.com/oauth/authorize",
			"token_endpoint":         "https://issuer.example.com/oauth/token",
			"userinfo_endpoint":      "https://issuer.example.com/userinfo",
		})
	}))
	t.Cleanup(server.Close)

	provider := New(
		WithName("generic"),
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithDiscoveryURL(server.URL),
		WithMapUserInfo(func(raw map[string]any) (*oauth.ProviderUserInfo, error) {
			return &oauth.ProviderUserInfo{ID: raw["sub"].(string), Email: raw["email"].(string)}, nil
		}),
	)

	cfg, _ := provider.OAuth2Config()
	if cfg.Endpoint.AuthURL != "https://issuer.example.com/oauth/authorize" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://issuer.example.com/oauth/token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
	if !provider.(oauth.IDTokenNonceProvider).IDTokenNonceEnabled() {
		t.Fatal("discovered issuer should enable ID token nonce checks")
	}
}

func TestExplicitEndpointsOverrideDiscovery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, map[string]any{
			"issuer":                 "https://issuer.example.com",
			"authorization_endpoint": "https://issuer.example.com/oauth/authorize",
			"token_endpoint":         "https://issuer.example.com/oauth/token",
			"userinfo_endpoint":      "https://issuer.example.com/userinfo",
		})
	}))
	t.Cleanup(server.Close)

	provider := New(
		WithName("generic"),
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithDiscoveryURL(server.URL),
		WithAuthorizationURL("https://override.example.com/authorize"),
		WithTokenURL("https://override.example.com/token"),
		WithUserInfoURL("https://override.example.com/userinfo"),
		WithMapUserInfo(func(raw map[string]any) (*oauth.ProviderUserInfo, error) {
			return &oauth.ProviderUserInfo{ID: raw["sub"].(string), Email: raw["email"].(string)}, nil
		}),
	)

	cfg, _ := provider.OAuth2Config()
	if cfg.Endpoint.AuthURL != "https://override.example.com/authorize" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://override.example.com/token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
}

func TestOAuth2Config_IncludesDefaultsRedirectAndOptions(t *testing.T) {
	t.Parallel()

	provider := New(
		WithName("generic"),
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithAuthorizationURL("https://provider.example.com/oauth/authorize"),
		WithTokenURL("https://provider.example.com/oauth/token"),
		WithRedirectURL("https://app.example/callback"),
		WithScopes(),
		WithOption("prompt", "login"),
		WithGetUserInfo(func(context.Context, *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
			return &oauth.ProviderUserInfo{ID: "id", Email: "user@example.com"}, nil
		}),
	)

	cfg, opts := provider.OAuth2Config()
	assertScopes(t, cfg.Scopes, "openid", "email", "profile")
	if cfg.RedirectURL != "https://app.example/callback" {
		t.Fatalf("RedirectURL = %q", cfg.RedirectURL)
	}
	if !strings.Contains(cfg.AuthCodeURL("state", opts...), "prompt=login") {
		t.Fatal("auth URL missing prompt option")
	}
}

func TestFetchUserInfoFromURL_RequestAndErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		mapper  func(map[string]any) (*oauth.ProviderUserInfo, error)
		wantErr bool
	}{
		{
			name:   "success",
			status: http.StatusOK,
			body:   `{"sub":"generic-user-1","email":"user@example.com","email_verified":true}`,
			mapper: func(raw map[string]any) (*oauth.ProviderUserInfo, error) {
				return &oauth.ProviderUserInfo{
					ID:            raw["sub"].(string),
					Email:         raw["email"].(string),
					EmailVerified: raw["email_verified"].(bool),
				}, nil
			},
		},
		{name: "non 200", status: http.StatusUnauthorized, body: `{}`, wantErr: true},
		{name: "bad json", status: http.StatusOK, body: `{`, wantErr: true},
		{
			name:   "mapper error",
			status: http.StatusOK,
			body:   `{"sub":"generic-user-1","email":"user@example.com"}`,
			mapper: func(map[string]any) (*oauth.ProviderUserInfo, error) {
				return nil, errors.New("map failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
					t.Fatalf("Authorization = %q", got)
				}
				if got := r.Header.Get("Accept"); got != "application/json" {
					t.Fatalf("Accept = %q", got)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			mapper := tt.mapper
			if mapper == nil {
				mapper = func(raw map[string]any) (*oauth.ProviderUserInfo, error) {
					return &oauth.ProviderUserInfo{ID: raw["sub"].(string), Email: raw["email"].(string)}, nil
				}
			}
			provider := New(
				WithName("generic"),
				WithClientID("client-id"),
				WithClientSecret("client-secret"),
				WithAuthorizationURL("https://provider.example.com/oauth/authorize"),
				WithTokenURL("https://provider.example.com/oauth/token"),
				WithUserInfoURL(server.URL),
				WithMapUserInfo(mapper),
			)

			info, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("GetUserInfo: %v", err)
			}
			if info.ID != "generic-user-1" || !info.EmailVerified {
				t.Fatalf("unexpected user info: %#v", info)
			}
		})
	}
}

func TestGetUserInfo_IDTokenOnlyErrors(t *testing.T) {
	t.Parallel()

	provider := New(
		WithName("generic"),
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithAuthorizationURL("https://provider.example.com/oauth/authorize"),
		WithTokenURL("https://provider.example.com/oauth/token"),
		WithMapUserInfo(func(raw map[string]any) (*oauth.ProviderUserInfo, error) {
			return &oauth.ProviderUserInfo{ID: raw["sub"].(string), Email: raw["email"].(string)}, nil
		}),
	)

	_, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{})
	if err == nil {
		t.Fatal("expected missing id_token error")
	}
}

func TestCustomExchangeAndRefreshFunctions(t *testing.T) {
	t.Parallel()

	var gotCode, gotVerifier, gotRedirect, gotRefresh string
	provider := New(
		WithName("generic"),
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithAuthorizationURL("https://provider.example.com/oauth/authorize"),
		WithExchangeTokens(func(_ context.Context, code, codeVerifier, redirectURI string) (*oauth.TokenResponse, error) {
			gotCode = code
			gotVerifier = codeVerifier
			gotRedirect = redirectURI
			return &oauth.TokenResponse{AccessToken: "access-token"}, nil
		}),
		WithRefreshTokens(func(_ context.Context, refreshToken string) (*oauth.TokenResponse, error) {
			gotRefresh = refreshToken
			return &oauth.TokenResponse{AccessToken: "new-access-token"}, nil
		}),
		WithGetUserInfo(func(context.Context, *oauth.TokenResponse) (*oauth.ProviderUserInfo, error) {
			return &oauth.ProviderUserInfo{ID: "id", Email: "user@example.com"}, nil
		}),
	)

	exchanger := provider.(oauth.TokenExchanger)
	token, err := exchanger.ExchangeAuthorizationCode(context.Background(), "code", "verifier", "https://app.example/callback")
	if err != nil {
		t.Fatalf("ExchangeAuthorizationCode: %v", err)
	}
	if token.AccessToken != "access-token" || gotCode != "code" || gotVerifier != "verifier" || gotRedirect != "https://app.example/callback" {
		t.Fatalf("unexpected exchange values")
	}

	refresher := provider.(oauth.TokenRefresher)
	token, err = refresher.RefreshToken(context.Background(), "refresh-token")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if token.AccessToken != "new-access-token" || gotRefresh != "refresh-token" {
		t.Fatalf("unexpected refresh values")
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

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
}
