package oauthtwitter

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/oauth2"

	"github.com/ragokan/limen/plugins/oauth"
)

func TestOAuth2Config_Defaults(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithRedirectURL("https://app.example/callback"),
		WithScopes(),
		WithOption("prompt", "consent"),
	).(*twitterProvider)

	cfg, opts := provider.OAuth2Config()
	if provider.Name() != "twitter" {
		t.Fatalf("Name = %q", provider.Name())
	}
	if cfg.Endpoint.AuthURL != "https://x.com/i/oauth2/authorize" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://api.x.com/2/oauth2/token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
	if cfg.Endpoint.AuthStyle != oauth2.AuthStyleInHeader {
		t.Fatalf("AuthStyle = %v", cfg.Endpoint.AuthStyle)
	}
	assertScopes(t, cfg.Scopes, "users.read", "users.email", "tweet.read", "offline.access")
	if !strings.Contains(cfg.AuthCodeURL("state", opts...), "prompt=consent") {
		t.Fatal("auth URL missing prompt option")
	}
}

func TestGetUserInfo_MapsConfirmedEmail(t *testing.T) {
	t.Parallel()

	provider := New().(*twitterProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := req.URL.String(); got != "https://api.x.com/2/users/me?user.fields=profile_image_url,confirmed_email" {
			t.Fatalf("URL = %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": "twitter-user-1",
					"name": "Test User",
					"username": "testuser",
					"profile_image_url": "https://example.com/avatar.png",
					"confirmed_email": "user@example.com"
				}
			}`)),
		}, nil
	})}

	info, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.ID != "twitter-user-1" || info.Email != "user@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected user info: %#v", info)
	}
}

func TestGetUserInfo_RequiresConfirmedEmail(t *testing.T) {
	t.Parallel()

	provider := New().(*twitterProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": "twitter-user-1",
					"username": "testuser"
				}
			}`)),
		}, nil
	})}

	_, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
	if err == nil {
		t.Fatal("expected missing email error")
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
