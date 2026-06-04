package oauthfacebook

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ragokan/limen/plugins/oauth"
)

func TestGetUserInfo_EmailVerificationUnknown(t *testing.T) {
	t.Parallel()

	provider := New().(*facebookProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q", got)
		}
		if got := req.URL.String(); got != facebookUserInfoURL {
			t.Fatalf("URL = %q, want %q", got, facebookUserInfoURL)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"id": "facebook-user-1",
				"name": "Test User",
				"email": "user@example.com",
				"picture": {"data": {"url": "https://example.com/avatar.png"}}
			}`)),
		}, nil
	})}

	info, err := provider.GetUserInfo(context.Background(), &oauth.TokenResponse{AccessToken: "access-token"})
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}
	if info.ID != "facebook-user-1" || info.Email != "user@example.com" {
		t.Fatalf("unexpected user info: %#v", info)
	}
	if info.EmailVerified {
		t.Fatalf("facebook email should not be marked verified: %#v", info)
	}
}

func TestOAuth2Config_UsesGraphV25EndpointsAndDefaultScopes(t *testing.T) {
	t.Parallel()

	provider := New(
		WithClientID("client-id"),
		WithClientSecret("client-secret"),
		WithRedirectURL("https://app.example/callback"),
		WithScopes(),
		WithOption("auth_type", "rerequest"),
	)
	cfg, opts := provider.OAuth2Config()

	if cfg.Endpoint.AuthURL != "https://www.facebook.com/v25.0/dialog/oauth" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://graph.facebook.com/v25.0/oauth/access_token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
	assertScopes(t, cfg.Scopes, "email", "public_profile")
	authURL := cfg.AuthCodeURL("state", opts...)
	if !strings.Contains(authURL, "auth_type=rerequest") {
		t.Fatalf("auth URL missing option: %s", authURL)
	}
}

func TestGetUserInfo_RejectsMissingID(t *testing.T) {
	t.Parallel()

	provider := New().(*facebookProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"name": "Test User",
				"email": "user@example.com"
			}`)),
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
