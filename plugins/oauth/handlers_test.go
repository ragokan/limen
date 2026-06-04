package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

func TestHandleCallbackResponseRedirects(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/oauth/test/callback", nil)

	t.Run("success redirect", func(t *testing.T) {
		t.Parallel()

		handlers := newOAuthHandlersForTest(t)
		rec := httptest.NewRecorder()

		handlers.handleCallbackResponse(rec, req, map[string]any{
			redirectURIKey: "http://localhost:8080/success",
		}, nil, nil, nil)

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "http://localhost:8080/success", rec.Header().Get("Location"))
	})

	t.Run("error redirect uses error_redirect_uri", func(t *testing.T) {
		t.Parallel()

		handlers := newOAuthHandlersForTest(t)
		rec := httptest.NewRecorder()

		handlers.handleCallbackResponse(rec, req, map[string]any{
			redirectURIKey:      "http://localhost:8080/success",
			errorRedirectURIKey: "http://localhost:8080/error",
		}, nil, nil, errors.New("bad callback"))

		assert.Equal(t, http.StatusSeeOther, rec.Code)
		location := rec.Header().Get("Location")
		assert.True(t, strings.HasPrefix(location, "http://localhost:8080/error?"))
		assert.Contains(t, location, "error=bad+callback")
	})

	t.Run("nil state returns json error", func(t *testing.T) {
		t.Parallel()

		handlers := newOAuthHandlersForTest(t)
		rec := httptest.NewRecorder()

		handlers.handleCallbackResponse(rec, req, nil, nil, nil, ErrOAuthStateInvalid)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Empty(t, rec.Header().Get("Location"))
		assert.Contains(t, rec.Body.String(), "invalid or expired OAuth state")
	})
}

func TestHandleCallbackResponseDisableRedirect(t *testing.T) {
	t.Parallel()

	l, plugin := newTestOAuthPlugin(t, WithDisableRedirect())
	_ = l.Handler()
	user := seedOAuthUser(t, l, "redirect-disabled@example.com")
	handlers := newOAuthHandlers(plugin, plugin.httpCore)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/oauth/test/callback", nil)
	rec := httptest.NewRecorder()

	handlers.handleCallbackResponse(rec, req, map[string]any{
		redirectURIKey: "http://localhost:8080/success",
	}, &limen.AuthenticationResult{User: user}, nil, nil)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("Location"))
	assert.Contains(t, rec.Body.String(), "redirect-disabled@example.com")
}

func TestFormPostCallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		requestURL       string
		body             string
		contentType      string
		wantStatus       int
		wantPath         string
		wantStoredParams map[string]string
		assertNoLocation bool
	}{
		{
			name:        "stores POST form params outside redirect URL",
			requestURL:  "/oauth/test/callback",
			body:        url.Values{"code": {"auth-code-123"}, "state": {"state-token-456"}}.Encode(),
			contentType: "application/x-www-form-urlencoded",
			wantStatus:  http.StatusSeeOther,
			wantPath:    "/oauth/test/callback",
			wantStoredParams: map[string]string{
				"code":  "auth-code-123",
				"state": "state-token-456",
			},
		},
		{
			name:        "stores error params from form body",
			requestURL:  "/oauth/test/callback",
			body:        url.Values{"state": {"state-token"}, "error": {"access_denied"}, "error_description": {"user canceled"}}.Encode(),
			contentType: "application/x-www-form-urlencoded",
			wantStatus:  http.StatusSeeOther,
			wantPath:    "/oauth/test/callback",
			wantStoredParams: map[string]string{
				"state":             "state-token",
				"error":             "access_denied",
				"error_description": "user canceled",
			},
		},
		{
			name:        "stores state-only form body",
			requestURL:  "/oauth/test/callback",
			body:        url.Values{"state": {"state-only"}}.Encode(),
			contentType: "application/x-www-form-urlencoded",
			wantStatus:  http.StatusSeeOther,
			wantPath:    "/oauth/test/callback",
			wantStoredParams: map[string]string{
				"state": "state-only",
			},
		},
		{
			name:        "stores existing query params and all form params",
			requestURL:  "/oauth/test/callback?client_hint=abc&foo=bar",
			body:        url.Values{"code": {"auth-code-123"}, "state": {"state-token-456"}, "custom_param": {"custom-value"}}.Encode(),
			contentType: "application/x-www-form-urlencoded",
			wantStatus:  http.StatusSeeOther,
			wantPath:    "/oauth/test/callback",
			wantStoredParams: map[string]string{
				"client_hint":  "abc",
				"foo":          "bar",
				"code":         "auth-code-123",
				"state":        "state-token-456",
				"custom_param": "custom-value",
			},
		},
		{
			name:             "returns error when form body is malformed",
			requestURL:       "/oauth/test/callback",
			body:             "state=%zz",
			contentType:      "application/x-www-form-urlencoded",
			wantStatus:       http.StatusInternalServerError,
			assertNoLocation: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handlers := newOAuthHandlersForTest(t)
			req := newFormPostRequest(t, tt.requestURL, tt.body, tt.contentType)
			rec := httptest.NewRecorder()

			handlers.FormPostCallback(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)

			location := rec.Header().Get("Location")
			if tt.assertNoLocation {
				assert.Empty(t, location)
				return
			}

			require.NotEmpty(t, location)
			parsed, err := url.Parse(location)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, parsed.Path)
			assert.Equal(t, "1", parsed.Query().Get(formPostQueryKey))
			for _, key := range []string{"code", "state", "error", "error_description", "custom_param", "user"} {
				assert.False(t, parsed.Query().Has(key), "sensitive callback param %q leaked into redirect URL", key)
			}

			params := decryptFormPostCookie(t, handlers, rec.Result().Cookies())
			for key, expected := range tt.wantStoredParams {
				assert.Equal(t, expected, params.Get(key))
			}
		})
	}
}

func TestCallbackParamsConsumesFormPostCookie(t *testing.T) {
	t.Parallel()

	handlers := newOAuthHandlersForTest(t)
	params := url.Values{"code": {"auth-code"}, "state": {"state-token"}}
	encrypted, err := limen.EncryptXChaCha(params.Encode(), handlers.plugin.config.secret, nil)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/oauth/test/callback?"+formPostQueryKey+"=1", nil)
	req.AddCookie(&http.Cookie{Name: formPostCookieName, Value: encrypted})
	rec := httptest.NewRecorder()

	got, err := handlers.callbackParams(rec, req)
	require.NoError(t, err)
	assert.Equal(t, "auth-code", got.Get("code"))
	assert.Equal(t, "state-token", got.Get("state"))
	assert.Contains(t, rec.Header().Values("Set-Cookie")[0], formPostCookieName+"=")
}

func decryptFormPostCookie(t *testing.T, handlers *oauthHandlers, cookies []*http.Cookie) url.Values {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name != formPostCookieName {
			continue
		}
		raw, err := limen.DecryptXChaCha(cookie.Value, handlers.plugin.config.secret, nil)
		require.NoError(t, err)
		params, err := url.ParseQuery(raw)
		require.NoError(t, err)
		return params
	}
	t.Fatalf("missing %s cookie", formPostCookieName)
	return nil
}

func newOAuthHandlersForTest(t *testing.T) *oauthHandlers {
	t.Helper()
	l, plugin := newTestOAuthPlugin(t)
	_ = l.Handler()
	return newOAuthHandlers(plugin, plugin.httpCore)
}

func newFormPostRequest(t *testing.T, requestURL, body, contentType string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, requestURL, strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	return req
}
