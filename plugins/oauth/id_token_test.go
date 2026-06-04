package oauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gojose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestVerifyIDTokenClaims_VerifiesDiscoveredJWKS(t *testing.T) {
	t.Parallel()

	issuer, token := newTestOIDCProviderAndToken(t, "client-id", time.Now().Add(time.Hour))

	claims, err := VerifyIDTokenClaims(t.Context(), issuer, "client-id", token)
	if err != nil {
		t.Fatalf("VerifyIDTokenClaims: %v", err)
	}
	if claims["sub"] != "user-1" || claims["email"] != "user@example.com" || claims["email_verified"] != true {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestVerifyIDTokenClaims_RejectsWrongAudience(t *testing.T) {
	t.Parallel()

	issuer, token := newTestOIDCProviderAndToken(t, "other-client-id", time.Now().Add(time.Hour))

	_, err := VerifyIDTokenClaims(t.Context(), issuer, "client-id", token)
	if err == nil {
		t.Fatal("expected wrong audience to be rejected")
	}
}

func TestVerifyIDTokenClaims_RejectsExpiredToken(t *testing.T) {
	t.Parallel()

	issuer, token := newTestOIDCProviderAndToken(t, "client-id", time.Now().Add(-time.Hour))

	_, err := VerifyIDTokenClaims(t.Context(), issuer, "client-id", token)
	if err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestVerifyIDTokenClaims_RejectsWrongIssuer(t *testing.T) {
	t.Parallel()

	issuer, token := newTestOIDCProviderAndTokenWithOptions(t, testOIDCTokenOptions{
		audience:  "client-id",
		expiresAt: time.Now().Add(time.Hour),
		issuer:    "https://issuer.example.invalid",
	})

	_, err := VerifyIDTokenClaims(t.Context(), issuer, "client-id", token)
	if err == nil {
		t.Fatal("expected wrong issuer to be rejected")
	}
}

func TestVerifyIDTokenClaims_RejectsUnknownKeyID(t *testing.T) {
	t.Parallel()

	issuer, token := newTestOIDCProviderAndTokenWithOptions(t, testOIDCTokenOptions{
		audience:  "client-id",
		expiresAt: time.Now().Add(time.Hour),
		keyID:     "unknown-key",
	})

	_, err := VerifyIDTokenClaims(t.Context(), issuer, "client-id", token)
	if err == nil {
		t.Fatal("expected unknown key id to be rejected")
	}
}

func TestVerifyIDTokenClaims_RejectsMalformedJWT(t *testing.T) {
	t.Parallel()

	issuer, _ := newTestOIDCProviderAndToken(t, "client-id", time.Now().Add(time.Hour))
	_, err := VerifyIDTokenClaims(t.Context(), issuer, "client-id", "not-a-jwt")
	if err == nil {
		t.Fatal("expected malformed JWT to be rejected")
	}
}

type testOIDCTokenOptions struct {
	audience  string
	expiresAt time.Time
	issuer    string
	keyID     string
}

func newTestOIDCProviderAndToken(t *testing.T, audience string, expiresAt time.Time) (string, string) {
	return newTestOIDCProviderAndTokenWithOptions(t, testOIDCTokenOptions{
		audience:  audience,
		expiresAt: expiresAt,
	})
}

func newTestOIDCProviderAndTokenWithOptions(t *testing.T, opts testOIDCTokenOptions) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	publicJWK := gojose.JSONWebKey{
		Key:       &privateKey.PublicKey,
		KeyID:     "test-key",
		Algorithm: string(gojose.RS256),
		Use:       "sig",
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			writeJSON(t, w, map[string]any{
				"issuer":                                server.URL,
				"jwks_uri":                              server.URL + "/jwks",
				"id_token_signing_alg_values_supported": []string{"RS256"},
			})
		case "/jwks":
			writeJSON(t, w, map[string]any{
				"keys": []gojose.JSONWebKey{publicJWK},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	keyID := opts.keyID
	if keyID == "" {
		keyID = "test-key"
	}
	issuer := opts.issuer
	if issuer == "" {
		issuer = server.URL
	}

	signer, err := gojose.NewSigner(
		gojose.SigningKey{Algorithm: gojose.RS256, Key: privateKey},
		(&gojose.SignerOptions{}).WithHeader("kid", keyID).WithType("JWT"),
	)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}

	token, err := jwt.Signed(signer).
		Claims(jwt.Claims{
			Issuer:   issuer,
			Subject:  "user-1",
			Audience: jwt.Audience{opts.audience},
			Expiry:   jwt.NewNumericDate(opts.expiresAt),
			IssuedAt: jwt.NewNumericDate(time.Now()),
		}).
		Claims(map[string]any{
			"email":          "user@example.com",
			"email_verified": true,
		}).
		Serialize()
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return server.URL, token
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
}
