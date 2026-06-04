package oauthconsentkeys

import "testing"

func TestOAuth2Config_Defaults(t *testing.T) {
	t.Parallel()

	provider := newConsentKeysProvider(&config{
		clientID:         "client-id",
		clientSecret:     "client-secret",
		redirectURL:      "https://app.example/callback",
		authorizationURL: "https://api.consentkeys.com/oauth/authorize",
		tokenURL:         "https://api.consentkeys.com/oauth/token",
		userInfoURL:      "https://api.consentkeys.com/userinfo",
	})

	cfg, _ := provider.OAuth2Config()
	if provider.Name() != "consentkeys" {
		t.Fatalf("Name = %q", provider.Name())
	}
	if cfg.Endpoint.AuthURL != "https://api.consentkeys.com/oauth/authorize" {
		t.Fatalf("AuthURL = %q", cfg.Endpoint.AuthURL)
	}
	if cfg.Endpoint.TokenURL != "https://api.consentkeys.com/oauth/token" {
		t.Fatalf("TokenURL = %q", cfg.Endpoint.TokenURL)
	}
	assertScopes(t, cfg.Scopes, "openid", "profile", "email")
}

func TestMapUserInfo(t *testing.T) {
	t.Parallel()

	info, err := mapUserInfo(map[string]any{
		"sub":                "consentkeys-user-1",
		"email":              "user@example.com",
		"email_verified":     "true",
		"preferred_username": "testuser",
		"picture":            "https://example.com/avatar.png",
	})
	if err != nil {
		t.Fatalf("mapUserInfo: %v", err)
	}
	if info.ID != "consentkeys-user-1" || info.Email != "user@example.com" || !info.EmailVerified {
		t.Fatalf("unexpected user info: %#v", info)
	}
	if info.Name != "testuser" || info.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("unexpected profile fields: %#v", info)
	}
}

func TestMapUserInfo_RejectsMissingSub(t *testing.T) {
	t.Parallel()

	_, err := mapUserInfo(map[string]any{"email": "user@example.com"})
	if err == nil {
		t.Fatal("expected missing sub error")
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
