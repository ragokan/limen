package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/ragokan/limen"
)

type callbackParamsContextKey struct{}
type idTokenNonceContextKey struct{}

// ContextWithCallbackParams returns a child context carrying the raw callback
// query parameters. Providers can retrieve them via CallbackParams inside
// GetUserInfo to access IdP-specific extras (e.g. Apple's first-login user payload).
func ContextWithCallbackParams(ctx context.Context, params url.Values) context.Context {
	return context.WithValue(ctx, callbackParamsContextKey{}, params)
}

// CallbackParams retrieves the callback query parameters stored in ctx, or nil.
func CallbackParams(ctx context.Context) url.Values {
	v, _ := ctx.Value(callbackParamsContextKey{}).(url.Values)
	return v
}

// ContextWithIDTokenNonce returns a child context carrying the expected OIDC nonce.
func ContextWithIDTokenNonce(ctx context.Context, nonce string) context.Context {
	return context.WithValue(ctx, idTokenNonceContextKey{}, nonce)
}

// IDTokenNonce retrieves the expected OIDC nonce stored in ctx, or an empty string.
func IDTokenNonce(ctx context.Context) string {
	v, _ := ctx.Value(idTokenNonceContextKey{}).(string)
	return v
}

// VerifyIDTokenNonce verifies the nonce claim against the expected OAuth state nonce.
// Apple may return the SHA-256 hex digest of the sent nonce; raw equality is accepted too.
func VerifyIDTokenNonce(claims map[string]any, expected string) error {
	if expected == "" {
		return fmt.Errorf("id token nonce is required")
	}
	claim, _ := claims["nonce"].(string)
	if claim == "" {
		return fmt.Errorf("id token missing nonce claim")
	}
	if subtle.ConstantTimeCompare([]byte(claim), []byte(expected)) == 1 {
		return nil
	}
	sum := sha256.Sum256([]byte(expected))
	digest := hex.EncodeToString(sum[:])
	if subtle.ConstantTimeCompare([]byte(claim), []byte(digest)) == 1 {
		return nil
	}
	return fmt.Errorf("id token nonce mismatch")
}

// BuildAuthCodeURL builds the OAuth2 authorization URL using the provider's config.
// state and verifier are required for CSRF and PKCE; authOpts add provider-specific params (e.g. AccessTypeOffline).
func BuildAuthCodeURL(config *oauth2.Config, state, verifier string, authOpts ...oauth2.AuthCodeOption) string {
	opts := make([]oauth2.AuthCodeOption, 0, len(authOpts)+2)
	opts = append(opts, authOpts...)
	if verifier != "" {
		opts = append(opts,
			oauth2.S256ChallengeOption(verifier),
		)
	}
	return config.AuthCodeURL(state, opts...)
}

// ExchangeCode exchanges an authorization code for tokens using the provider's config.
// codeVerifier is required when PKCE was used on the authorization URL.
func ExchangeCode(ctx context.Context, config *oauth2.Config, code, codeVerifier string) (*TokenResponse, error) {
	var opts []oauth2.AuthCodeOption
	if codeVerifier != "" {
		opts = append(opts, oauth2.VerifierOption(codeVerifier))
	}
	tok, err := config.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, err
	}

	resp := &TokenResponse{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		ExpiresAt:    tok.Expiry,
	}

	if extra, ok := tok.Extra("id_token").(string); ok {
		resp.IDToken = extra
	}
	if scope, ok := tok.Extra("scope").(string); ok && scope != "" {
		resp.Scope = scope
	}
	if resp.Scope == "" {
		resp.Scope = strings.Join(config.Scopes, ",")
	}
	return resp, nil
}

// RefreshToken uses the standard oauth2.TokenSource to exchange a refresh token
// for a new access token via the provider's token endpoint.
func RefreshToken(ctx context.Context, config *oauth2.Config, refreshToken string) (*TokenResponse, error) {
	src := config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	tok, err := src.Token()
	if err != nil {
		return nil, err
	}

	resp := &TokenResponse{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		ExpiresAt:    tok.Expiry,
	}

	if extra, ok := tok.Extra("id_token").(string); ok {
		resp.IDToken = extra
	}
	if scope, ok := tok.Extra("scope").(string); ok && scope != "" {
		resp.Scope = scope
	}
	return resp, nil
}

// FetchUserInfoJSON performs a GET request to the given URL with a Bearer token
// and decodes the JSON response into a map. Shared by REST-based OAuth providers.
func FetchUserInfoJSON(ctx context.Context, client *http.Client, providerName, endpointURL, accessToken string, extraHeaders map[string]string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: user info request failed: %s", providerName, resp.Status)
	}

	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// DecodeIDTokenClaims decodes the payload segment of a JWT without verification.
//
// NOTE: This does not verify the token, so it is not safe to use for any purpose other than to get the claims
// from the id_token returned by the provider.
func DecodeIDTokenClaims(idToken string) (map[string]any, error) {
	parts := strings.SplitN(idToken, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("id_token has invalid JWT format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("id_token payload decode: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("id_token payload unmarshal: %w", err)
	}
	return claims, nil
}

// generateCodeVerifier creates a cryptographically random PKCE code_verifier
func generateCodeVerifier() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("oauth: crypto random read failed: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// generateRandomString generates a cryptographically secure random string
func generateRandomString() string {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		panic(fmt.Sprintf("oauth: crypto random read failed: %v", err))
	}
	return hex.EncodeToString(randomBytes)
}

func newAccountFromOAuthProfile(userID any, profile *limen.OAuthAccountProfile, tokens *OAuthTokens) *limen.Account {
	now := time.Now()
	return &limen.Account{
		UserID:               userID,
		Provider:             profile.Provider,
		ProviderAccountID:    profile.ProviderAccountID,
		AccessToken:          tokens.AccessToken,
		RefreshToken:         tokens.RefreshToken,
		AccessTokenExpiresAt: profile.AccessTokenExpiresAt,
		Scope:                profile.Scope,
		IDToken:              tokens.IDToken,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}
