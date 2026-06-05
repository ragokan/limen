package sessionjwt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ragokan/limen"
)

var testJWTSecret = []byte("test-secret-key-for-jwt-signing!")

func newTestPlugin() *sessionJWTPlugin {
	return &sessionJWTPlugin{
		config: &config{
			signingMethod:        jwt.SigningMethodHS256,
			signingKey:           testJWTSecret,
			verificationKey:      testJWTSecret,
			accessTokenDuration:  15 * time.Minute,
			refreshTokenDuration: 7 * 24 * time.Hour,
			refreshTokenRotation: true,
			refreshTokenEnabled:  true,
			issuer:               "test-issuer",
			audience:             []string{"test-audience"},
			subjectEncoder:       func(user *limen.User) string { return fmt.Sprintf("%v", user.ID) },
			subjectResolver:      func(subject string) (any, error) { return subject, nil },
		},
	}
}

func TestGenerateAccessToken(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	user := &limen.User{ID: "user-1", Email: "a@b.com"}

	signed, jti, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, signed)
	assert.NotEmpty(t, jti)
}

func TestGenerateAccessToken_ContainsClaims(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	user := &limen.User{ID: "user-1", Email: "a@b.com"}

	signed, _, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)

	claims, err := plugin.VerifyAccessToken(signed)
	assert.NoError(t, err)
	assert.Equal(t, "user-1", claims.Subject)
	assert.Equal(t, "a@b.com", claims.Email)
	assert.Equal(t, "test-issuer", claims.Issuer)
	assert.Contains(t, claims.Audience, "test-audience")
}

func TestGenerateAccessToken_CustomClaims(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	plugin.config.customClaims = func(user *limen.User) map[string]any {
		return map[string]any{"role": "admin"}
	}

	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)

	claims, err := plugin.VerifyAccessToken(signed)
	assert.NoError(t, err)
	assert.Equal(t, "admin", claims.Custom["role"])
}

func TestVerifyAccessToken_Expired(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	plugin.config.accessTokenDuration = -1 * time.Second // already expired

	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)

	_, err = plugin.VerifyAccessToken(signed)
	assert.ErrorIs(t, err, ErrInvalidAccessToken)
}

func TestVerifyAccessToken_WrongSignature(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, err := plugin.GenerateAccessToken(user)
	assert.NoError(t, err)

	wrongPlugin := newTestPlugin()
	wrongPlugin.config.signingKey = []byte("completely-different-key-32bytes!")
	wrongPlugin.config.verificationKey = []byte("completely-different-key-32bytes!")

	_, err = wrongPlugin.VerifyAccessToken(signed)
	assert.ErrorIs(t, err, ErrInvalidAccessToken)
}

func TestVerifyAccessToken_InvalidToken(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()

	_, err := plugin.VerifyAccessToken("not.a.valid.token")
	assert.ErrorIs(t, err, ErrInvalidAccessToken)
}

func TestVerifyAccessToken_WrongIssuer(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, _ := plugin.GenerateAccessToken(user)

	verifier := newTestPlugin()
	verifier.config.issuer = "wrong-issuer"

	_, err := verifier.VerifyAccessToken(signed)
	assert.ErrorIs(t, err, ErrInvalidAccessToken)
}

func TestParseAccessTokenLenient_ExpiredButValid(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()
	plugin.config.accessTokenDuration = -1 * time.Second

	user := &limen.User{ID: "user-1", Email: "a@b.com"}
	signed, _, _ := plugin.GenerateAccessToken(user)

	claims := plugin.parseAccessTokenLenient(signed)
	assert.NotNil(t, claims)
	assert.Equal(t, "user-1", claims.Subject)
}

func TestPerformRefresh_MissingTokenWithFamilyAndNoActiveTokens(t *testing.T) {
	t.Parallel()

	plugin := New()
	limen.NewTestLimen(t, plugin)

	_, _, err := plugin.performRefresh(t.Context(), "missing-refresh-token", "family-1")

	assert.ErrorIs(t, err, ErrInvalidRefreshToken)
}

func TestPerformRefresh_MissingTokenWithActiveFamilyRevokesFamily(t *testing.T) {
	t.Parallel()

	plugin := New()
	limen.NewTestLimen(t, plugin)
	_, err := plugin.CreateRefreshToken(t.Context(), "user-1", "jti-1", "family-1", nil)
	require.NoError(t, err)

	_, _, err = plugin.performRefresh(t.Context(), "missing-refresh-token", "family-1")

	assert.ErrorIs(t, err, ErrRefreshTokenReuse)
	active, err := plugin.FamilyHasActiveTokens(t.Context(), "family-1")
	require.NoError(t, err)
	assert.False(t, active)
}

func TestListSessionsHTTPRedactsRefreshTokens(t *testing.T) {
	t.Parallel()

	plugin := New(WithSubjectResolver(func(subject string) (any, error) {
		return strconv.ParseInt(subject, 10, 64)
	}))
	auth, _ := limen.NewTestLimen(t, plugin)
	user := limen.SeedTestUser(t, auth, "jwt-sessions@test.com")
	session := limen.SeedTestSession(t, auth, user.ID, user.Email)
	require.NotEmpty(t, session.RefreshToken)
	refreshToken, _, _ := strings.Cut(session.RefreshToken, ".")
	require.NotEmpty(t, refreshToken)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/sessions", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+session.Token)
	resp := httptest.NewRecorder()
	auth.Handler().ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code, resp.Body.String())
	assert.NotContains(t, resp.Body.String(), session.RefreshToken)
	assert.NotContains(t, resp.Body.String(), refreshToken)
	assert.NotContains(t, resp.Body.String(), "token")
	assert.NotContains(t, resp.Body.String(), "refreshToken")

	var payload []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Len(t, payload, 1)
	assert.NotContains(t, payload[0], "token")
	assert.NotContains(t, payload[0], "refreshToken")
}

func TestRotateRefreshToken_NilOldToken(t *testing.T) {
	t.Parallel()

	plugin := newTestPlugin()

	_, err := plugin.RotateRefreshToken(t.Context(), nil, "new-jti")

	assert.ErrorIs(t, err, ErrInvalidRefreshToken)
}
