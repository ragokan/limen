package oauthspotify

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ragokan/limen/plugins/oauth"
)

func TestGetUserInfo_EmailIsNotVerified(t *testing.T) {
	t.Parallel()

	provider := New().(*spotifyProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := req.URL.String(); got != "https://api.spotify.com/v1/me" {
			t.Fatalf("URL = %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"id": "spotify-user-1",
				"display_name": "Test User",
				"email": "user@example.com",
				"email_verified": true,
				"images": [{"url": "https://example.com/avatar.png"}]
			}`)),
		}, nil
	})}

	info, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.ID != "spotify-user-1" || info.Email != "user@example.com" {
		t.Fatalf("unexpected user info: %#v", info)
	}
	if info.EmailVerified {
		t.Fatalf("spotify email should not be marked verified: %#v", info)
	}
}

func TestOAuth2Config_DefaultScopes(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithRedirectURL("https://app.example/callback"),
		WithScopes(),
		WithOption("show_dialog", "true"),
	).(*spotifyProvider)
	cfg, _ := provider.OAuth2Config()

	if cfg.Endpoint.AuthURL != "https://accounts.spotify.com/authorize" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://accounts.spotify.com/api/token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
	if len(cfg.Scopes) != 1 || cfg.Scopes[0] != "user-read-email" {
		t.Fatalf("Scopes = %#v, want user-read-email", cfg.Scopes)
	}
	_, opts := provider.OAuth2Config()
	authURL := cfg.AuthCodeURL("state", opts...)
	if !strings.Contains(authURL, "show_dialog=true") {
		t.Fatalf("auth URL missing option: %s", authURL)
	}
}

func TestGetUserInfo_MissingEmailReturnsError(t *testing.T) {
	t.Parallel()

	provider := New().(*spotifyProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"id": "spotify-user-1",
				"display_name": "Test User"
			}`)),
		}, nil
	})}

	_, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
	if err == nil {
		t.Fatal("expected missing email error")
	}
}

func TestGetUserInfo_MissingIDReturnsError(t *testing.T) {
	t.Parallel()

	provider := New().(*spotifyProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"display_name": "Test User",
				"email": "user@example.com"
			}`)),
		}, nil
	})}

	_, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
	if err == nil {
		t.Fatal("expected missing id error")
	}
}

func TestGetUserInfo_DisplayNameFallbackAndAvatar(t *testing.T) {
	t.Parallel()

	provider := New().(*spotifyProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"id": "spotify-user-1",
				"email": "user@example.com",
				"images": [{"url": "https://example.com/avatar.png"}]
			}`)),
		}, nil
	})}

	info, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.Name != "spotify-user-1" {
		t.Fatalf("Name = %q", info.Name)
	}
	if info.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("AvatarURL = %q", info.AvatarURL)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
