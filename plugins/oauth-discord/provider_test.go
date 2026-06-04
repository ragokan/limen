package oauthdiscord

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ragokan/limen/plugins/oauth"
)

func TestOAuth2Config_Defaults(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithRedirectURL("https://app.example/callback"),
		WithScopes(),
		WithOption("prompt", "none"),
	).(*discordProvider)

	cfg, opts := provider.OAuth2Config()
	if provider.Name() != "discord" {
		t.Fatalf("Name = %q", provider.Name())
	}
	if cfg.Endpoint.AuthURL != "https://discord.com/oauth2/authorize" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://discord.com/api/oauth2/token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
	assertScopes(t, cfg.Scopes, "identify", "email")
	if !strings.Contains(cfg.AuthCodeURL("state", opts...), "prompt=none") {
		t.Fatal("auth URL missing prompt option")
	}
}

func TestGetUserInfo_MapsProfile(t *testing.T) {
	t.Parallel()

	provider := New().(*discordProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := req.URL.String(); got != "https://discord.com/api/users/@me" {
			t.Fatalf("URL = %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"id": "discord-user-1",
				"username": "testuser",
				"email": "user@example.com",
				"verified": true,
				"avatar": "avatar-hash"
			}`)),
		}, nil
	})}

	info, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.ID != "discord-user-1" || info.Email != "user@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected user info: %#v", info)
	}
	if info.AvatarURL != "https://cdn.discordapp.com/avatars/discord-user-1/avatar-hash.png" {
		t.Fatalf("AvatarURL = %q", info.AvatarURL)
	}
}

func TestGetUserInfo_RejectsMissingID(t *testing.T) {
	t.Parallel()

	provider := New().(*discordProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"email":"user@example.com"}`)),
		}, nil
	})}

	_, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
	if err == nil {
		t.Fatal("expected missing id error")
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
