package oauth

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

type pkceDisabledProvider struct {
	testProvider
}

func (p *pkceDisabledProvider) PKCEEnabled() bool { return false }

type tokenExchangerProvider struct {
	testProvider
	lastCode        string
	lastVerifier    string
	lastRedirectURI string
	token           *TokenResponse
}

func (p *tokenExchangerProvider) ExchangeAuthorizationCode(_ context.Context, code, codeVerifier, redirectURI string) (*TokenResponse, error) {
	p.lastCode = code
	p.lastVerifier = codeVerifier
	p.lastRedirectURI = redirectURI
	return p.token, nil
}

type responseModeProvider struct {
	testProvider
}

func (p *responseModeProvider) ResponseMode() ResponseMode {
	return ResponseModeFormPost
}

type nonceProvider struct {
	testProvider
}

func (p *nonceProvider) IDTokenNonceEnabled() bool {
	return true
}

func TestGetAuthorizationURL(t *testing.T) {
	t.Parallel()

	t.Run("provider not found", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t)
		_ = l.Handler()

		authURL, cookieValue, err := plugin.GetAuthorizationURL(context.Background(), "missing", &OAuthAuthorizeURLData{})
		assert.Empty(t, authURL)
		assert.Empty(t, cookieValue)
		assert.ErrorIs(t, err, ErrProviderNotFound)
	})

	t.Run("untrusted redirect URI", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t)
		_ = l.Handler()

		authURL, cookieValue, err := plugin.GetAuthorizationURL(context.Background(), "test", &OAuthAuthorizeURLData{
			RedirectURI: "https://evil.example/callback",
		})
		assert.Empty(t, authURL)
		assert.Empty(t, cookieValue)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, limen.ToLimenError(err).Status())
	})

	t.Run("untrusted error redirect URI", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t)
		_ = l.Handler()

		authURL, cookieValue, err := plugin.GetAuthorizationURL(context.Background(), "test", &OAuthAuthorizeURLData{
			ErrorRedirectURI: "https://evil.example/error",
		})
		assert.Empty(t, authURL)
		assert.Empty(t, cookieValue)
		require.Error(t, err)
		assert.Equal(t, http.StatusForbidden, limen.ToLimenError(err).Status())
	})

	t.Run("provider can disable PKCE verifier", func(t *testing.T) {
		t.Parallel()

		provider := &pkceDisabledProvider{
			testProvider: testProvider{name: "pkce-disabled"},
		}
		l, plugin := newTestOAuthPlugin(t, WithProviders(provider))
		_ = l.Handler()

		authURL, cookieValue, err := plugin.GetAuthorizationURL(context.Background(), "pkce-disabled", &OAuthAuthorizeURLData{})
		require.NoError(t, err)
		assert.NotEmpty(t, authURL)
		assert.NotEmpty(t, cookieValue)

		parsed, parseErr := url.Parse(authURL)
		require.NoError(t, parseErr)
		assert.Empty(t, parsed.Query().Get("code_challenge"))
		assert.Empty(t, parsed.Query().Get("code_challenge_method"))
	})

	t.Run("form_post provider includes response_mode param", func(t *testing.T) {
		t.Parallel()

		formPostProvider := &responseModeProvider{
			testProvider: testProvider{name: "form-post"},
		}
		l, plugin := newTestOAuthPlugin(t, WithProviders(formPostProvider))
		_ = l.Handler()

		authURL, cookieValue, err := plugin.GetAuthorizationURL(context.Background(), "form-post", &OAuthAuthorizeURLData{})
		require.NoError(t, err)
		assert.NotEmpty(t, cookieValue)

		parsed, parseErr := url.Parse(authURL)
		require.NoError(t, parseErr)
		assert.Equal(t, "form_post", parsed.Query().Get("response_mode"))
		assert.Equal(t, "code", parsed.Query().Get("response_type"))
	})

	t.Run("returns state token in authorization URL", func(t *testing.T) {
		t.Parallel()

		l, plugin := newTestOAuthPlugin(t)
		_ = l.Handler()

		authURL, cookieValue, err := plugin.GetAuthorizationURL(context.Background(), "test", &OAuthAuthorizeURLData{})
		require.NoError(t, err)
		assert.NotEmpty(t, cookieValue)

		parsed, parseErr := url.Parse(authURL)
		require.NoError(t, parseErr)
		assert.NotEmpty(t, parsed.Query().Get("state"))
	})

	t.Run("nonce provider includes nonce param and stores expected nonce", func(t *testing.T) {
		t.Parallel()

		provider := &nonceProvider{testProvider: testProvider{name: "nonce-provider"}}
		l, plugin := newTestOAuthPlugin(t, WithProviders(provider))
		_ = l.Handler()

		authURL, cookieValue, err := plugin.GetAuthorizationURL(context.Background(), "nonce-provider", &OAuthAuthorizeURLData{})
		require.NoError(t, err)
		require.NotEmpty(t, cookieValue)

		parsed, parseErr := url.Parse(authURL)
		require.NoError(t, parseErr)
		nonce := parsed.Query().Get("nonce")
		stateToken := parsed.Query().Get("state")
		require.NotEmpty(t, nonce)
		require.NotEmpty(t, stateToken)

		stateData, err := plugin.stateStore.Validate(context.Background(), stateToken, cookieValue)
		require.NoError(t, err)
		assert.Equal(t, nonce, stateData[nonceDataKey])
	})
}

func TestExchangeAuthorizationCodeForTokens(t *testing.T) {
	t.Parallel()

	customProvider := &tokenExchangerProvider{
		testProvider: testProvider{name: "custom-exchange"},
		token: &TokenResponse{
			AccessToken:  "access",
			RefreshToken: "refresh",
			Scope:        "openid email",
		},
	}
	_, plugin := newTestOAuthPlugin(t, WithProviders(customProvider))

	t.Run("missing PKCE verifier", func(t *testing.T) {
		t.Parallel()

		token, err := plugin.ExchangeAuthorizationCodeForTokens(context.Background(), customProvider, map[string]any{}, "auth-code")
		assert.Nil(t, token)
		assert.ErrorIs(t, err, ErrPKCEVerifierNotFound)
	})

	t.Run("uses token exchanger", func(t *testing.T) {
		t.Parallel()

		stateData := map[string]any{pkceDataKey: "pkce-verifier"}
		token, err := plugin.ExchangeAuthorizationCodeForTokens(context.Background(), customProvider, stateData, "auth-code")
		require.NoError(t, err)
		require.NotNil(t, token)
		assert.Equal(t, "access", token.AccessToken)
		assert.Equal(t, "auth-code", customProvider.lastCode)
		assert.Equal(t, "pkce-verifier", customProvider.lastVerifier)
		assert.Contains(t, customProvider.lastRedirectURI, "/oauth/custom-exchange/callback")
	})
}

func TestHandleOAuthCallback(t *testing.T) {
	t.Parallel()

	t.Run("rejects provider mismatch", func(t *testing.T) {
		t.Parallel()

		for _, tt := range []struct {
			name string
			opts []ConfigOption
		}{
			{name: "stateless"},
			{name: "database", opts: []ConfigOption{WithDatabaseState()}},
		} {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				providers := []Provider{
					&testProvider{name: "test"},
					&testProvider{name: "other"},
				}
				opts := append([]ConfigOption{WithProviders(providers...)}, tt.opts...)
				l, plugin := newTestOAuthPlugin(t, opts...)
				_ = l.Handler()

				authURL, cookieValue, err := plugin.GetAuthorizationURL(context.Background(), "test", &OAuthAuthorizeURLData{})
				require.NoError(t, err)
				parsed, parseErr := url.Parse(authURL)
				require.NoError(t, parseErr)

				profile, stateData, callbackErr := plugin.HandleOAuthCallback(
					context.Background(),
					"other",
					"auth-code",
					parsed.Query().Get("state"),
					cookieValue,
					nil,
				)
				assert.Nil(t, profile)
				require.Error(t, callbackErr)
				assert.ErrorIs(t, callbackErr, ErrOAuthProviderMismatch)
				assert.Equal(t, "test", stateData[providerDataKey])
			})
		}
	})

	t.Run("callback error preserves state data", func(t *testing.T) {
		t.Parallel()

		_, plugin := newTestOAuthPlugin(t)
		ctx := context.Background()
		seedData := map[string]any{providerDataKey: "test", "source": "oauth-test"}
		stateToken, cookieNonce, err := plugin.stateStore.Generate(ctx, seedData)
		require.NoError(t, err)

		profile, stateData, callbackErr := plugin.HandleOAuthCallback(ctx, "test", "", stateToken, cookieNonce, &CallbackError{
			Code:        "access_denied",
			Description: "user canceled",
		})
		assert.Nil(t, profile)
		require.Error(t, callbackErr)
		assert.Equal(t, "oauth-test", stateData["source"])
		details, ok := limen.ToLimenError(callbackErr).Details().(map[string]string)
		require.True(t, ok)
		assert.Equal(t, "access_denied", details["code"])
	})

	t.Run("missing code after valid state", func(t *testing.T) {
		t.Parallel()

		_, plugin := newTestOAuthPlugin(t)
		ctx := context.Background()
		stateToken, cookieNonce, err := plugin.stateStore.Generate(ctx, map[string]any{
			providerDataKey: "test",
			"source":        "missing-code",
		})
		require.NoError(t, err)

		profile, stateData, callbackErr := plugin.HandleOAuthCallback(ctx, "test", "", stateToken, cookieNonce, nil)
		assert.Nil(t, profile)
		require.Error(t, callbackErr)
		assert.Equal(t, http.StatusBadRequest, limen.ToLimenError(callbackErr).Status())
		assert.Equal(t, "missing-code", stateData["source"])
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		customProvider := &tokenExchangerProvider{
			testProvider: testProvider{name: "cb-success"},
			token: &TokenResponse{
				AccessToken:  "access-token-success",
				RefreshToken: "refresh-token-success",
				Scope:        "openid email",
			},
		}
		l, plugin := newTestOAuthPlugin(t, WithProviders(customProvider))
		_ = l.Handler()

		authURL, cookieValue, err := plugin.GetAuthorizationURL(context.Background(), "cb-success", &OAuthAuthorizeURLData{})
		require.NoError(t, err)
		require.NotEmpty(t, cookieValue)

		parsed, parseErr := url.Parse(authURL)
		require.NoError(t, parseErr)
		stateToken := parsed.Query().Get("state")
		require.NotEmpty(t, stateToken)

		profile, stateData, callbackErr := plugin.HandleOAuthCallback(context.Background(), "cb-success", "auth-code", stateToken, cookieValue, nil)
		require.NoError(t, callbackErr)
		require.NotNil(t, profile)
		assert.Equal(t, "cb-success", profile.Provider)
		assert.Equal(t, "provider-user-123", profile.ProviderAccountID)
		assert.Equal(t, "oauth@example.com", profile.Email)
		assert.Equal(t, "access-token-success", profile.AccessToken)
		assert.Equal(t, "refresh-token-success", profile.RefreshToken)
		assert.Equal(t, "openid email", profile.Scope)
		pkceVerifier, ok := stateData[pkceDataKey].(string)
		require.True(t, ok)
		assert.NotEmpty(t, pkceVerifier)
	})
}
