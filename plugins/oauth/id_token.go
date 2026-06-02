package oauth

import (
	"context"
	"fmt"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
)

type IDTokenVerifier func(ctx context.Context, idToken string) (map[string]any, error)

func NewIDTokenVerifier(issuer, clientID string) IDTokenVerifier {
	var mu sync.Mutex
	var verifier *oidc.IDTokenVerifier

	return func(ctx context.Context, idToken string) (map[string]any, error) {
		mu.Lock()
		if verifier == nil {
			created, err := newOIDCIDTokenVerifier(ctx, issuer, clientID)
			if err != nil {
				mu.Unlock()
				return nil, err
			}
			verifier = created
		}
		current := verifier
		mu.Unlock()

		return verifyIDTokenClaims(ctx, current, idToken)
	}
}

func VerifyIDTokenClaims(ctx context.Context, issuer, clientID, idToken string) (map[string]any, error) {
	verifier, err := newOIDCIDTokenVerifier(ctx, issuer, clientID)
	if err != nil {
		return nil, err
	}
	return verifyIDTokenClaims(ctx, verifier, idToken)
}

func newOIDCIDTokenVerifier(ctx context.Context, issuer, clientID string) (*oidc.IDTokenVerifier, error) {
	if issuer == "" {
		return nil, fmt.Errorf("id token issuer is required")
	}
	if clientID == "" {
		return nil, fmt.Errorf("id token client ID is required")
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("id token provider discovery failed: %w", err)
	}

	return provider.Verifier(&oidc.Config{ClientID: clientID}), nil
}

func verifyIDTokenClaims(ctx context.Context, verifier *oidc.IDTokenVerifier, idToken string) (map[string]any, error) {
	verified, err := verifier.Verify(ctx, idToken)
	if err != nil {
		return nil, fmt.Errorf("id token verification failed: %w", err)
	}

	var claims map[string]any
	if err := verified.Claims(&claims); err != nil {
		return nil, fmt.Errorf("id token claims decode failed: %w", err)
	}
	return claims, nil
}
