package oauthspotify

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/thecodearcher/limen/plugins/oauth"
)

func TestGetUserInfo_EmailIsNotVerified(t *testing.T) {
	t.Parallel()

	provider := New().(*spotifyProvider)
	provider.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "Bearer access-token" {
			t.Fatalf("Authorization = %q", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body: io.NopCloser(strings.NewReader(`{
				"id": "spotify-user-1",
				"display_name": "Test User",
				"email": "user@example.com",
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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
