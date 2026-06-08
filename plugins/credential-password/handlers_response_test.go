package credentialpassword

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
	sessionjwt "github.com/ragokan/limen/plugins/session-jwt"
)

func TestCredentialHandlersSessionResponseIncludesTokensWithJWT(t *testing.T) {
	t.Parallel()

	auth, _ := limen.NewTestLimen(t, New(), sessionjwt.New())
	handler := auth.Handler()

	signUp := postJSON(t, handler, "/auth/signup/credential", map[string]any{
		"email":    "tokens@example.com",
		"password": "Password1",
	})
	require.Equal(t, http.StatusOK, signUp.Code, signUp.Body.String())
	assertSessionResponseTokens(t, signUp)

	signIn := postJSON(t, handler, "/auth/signin/credential", map[string]any{
		"credential": "tokens@example.com",
		"password":   "Password1",
	})
	require.Equal(t, http.StatusOK, signIn.Code, signIn.Body.String())
	assertSessionResponseTokens(t, signIn)
}

func postJSON(t *testing.T, handler http.Handler, path string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}

func assertSessionResponseTokens(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()

	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Contains(t, payload, "user")

	tokens, ok := payload["tokens"].(map[string]any)
	require.True(t, ok)
	authToken, _ := tokens["auth_token"].(string)
	refreshToken, _ := tokens["refresh_token"].(string)
	assert.NotEmpty(t, authToken)
	assert.NotEmpty(t, refreshToken)
}
