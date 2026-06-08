package limen

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestResponder(t *testing.T) *Responder {
	t.Helper()
	cfg := NewDefaultHTTPConfig()
	cm := newCookieManager(cfg.cookieConfig, testSecret)
	return newResponder(cfg, cm, false)
}

func TestResponder_JSON(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	responder.JSON(w, req, http.StatusOK, map[string]any{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), `"key":"value"`)
}

func TestResponder_JSON_StringMessage(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	responder.JSON(w, req, http.StatusOK, "success")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"success"`)
}

func TestResponderJSONNoBodyStatuses(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)

	for _, status := range []int{http.StatusNoContent, http.StatusNotModified} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()

			err := responder.JSON(w, req, status, nil)

			assert.NoError(t, err)
			assert.Equal(t, status, w.Code)
			assert.Empty(t, w.Body.String())
			assert.Empty(t, w.Header().Get("Content-Type"))
		})
	}
}

func TestResponder_Error(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	responder.Error(w, req, NewLimenError("something went wrong", http.StatusBadRequest, nil))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, w.Body.String(), `"message":"something went wrong"`)
}

func TestResponder_Error_GenericError(t *testing.T) {
	t.Parallel()

	responder := newTestResponder(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	w := httptest.NewRecorder()

	responder.Error(w, req, errors.New("generic error"))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"generic error"`)
}

func TestResponder_SessionResponseIncludesTokensWhenReturnedToClient(t *testing.T) {
	t.Parallel()

	l := newTestLimenWithSessionConfig(t, WithBearerEnabled())
	responder := newResponder(l.config.HTTP, l.core.cookies, true)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/signin", http.NoBody)
	w := httptest.NewRecorder()

	err := responder.SessionResponse(w, req, l.core, &AuthenticationResult{User: &User{
		ID:    "user-1",
		Email: "user@example.com",
	}}, &SessionResult{
		Token:        "auth-token",
		RefreshToken: "refresh-token",
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&payload))
	require.Contains(t, payload, "user")
	tokens, ok := payload["tokens"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "auth-token", tokens["auth_token"])
	assert.Equal(t, "refresh-token", tokens["refresh_token"])
}

func TestResponder_SessionResponseOmitsCookieOnlyTokenFromBody(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)
	responder := newResponder(l.config.HTTP, l.core.cookies, false)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/auth/signin", http.NoBody)
	w := httptest.NewRecorder()

	err := responder.SessionResponse(w, req, l.core, &AuthenticationResult{User: &User{
		ID:    "user-1",
		Email: "user@example.com",
	}}, &SessionResult{
		Token: "cookie-token",
		Cookie: &http.Cookie{
			Name:  "limen_session",
			Value: "cookie-token",
		},
	})

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&payload))
	require.Contains(t, payload, "user")
	assert.NotContains(t, payload, "tokens")
}

func TestToLimenError_LimenError(t *testing.T) {
	t.Parallel()

	original := NewLimenError("bad request", http.StatusBadRequest, nil)
	result := ToLimenError(original)

	assert.Equal(t, http.StatusBadRequest, result.Status())
	assert.Equal(t, "bad request", result.Error())
}

func TestToLimenError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "Returns Limen Errors As Is", err: ErrRecordNotFound, want: ErrRecordNotFound.Status()},
		{name: "Returns Generic Errors As Internal Server Error", err: errors.New("something"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToLimenError(tt.err)
			assert.Equal(t, tt.want, result.Status())
		})
	}
}
