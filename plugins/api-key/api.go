package apikey

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ragokan/limen"
)

func (p *apiKeyPlugin) CreateAPIKey(ctx context.Context, userID any, name string, opts ...CreateAPIKeyOption) (*CreatedAPIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, limen.NewLimenError("name is required", http.StatusUnprocessableEntity, nil)
	}

	options := &CreateAPIKeyOptions{}
	for _, opt := range opts {
		opt(options)
	}
	expiresAt := options.ExpiresAt
	if expiresAt == nil && p.config.defaultTTL > 0 {
		t := time.Now().Add(p.config.defaultTTL)
		expiresAt = &t
	}

	rawKey, err := p.generateKey()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	scopes, err := p.validateRequestedScopes(options.Scopes)
	if err != nil {
		return nil, err
	}

	apiKey := &APIKey{
		UserID:    userID,
		Name:      name,
		Prefix:    p.storedPrefix(rawKey),
		keyHash:   p.hashKey(rawKey),
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := p.core.Create(ctx, p.schema, apiKey, nil); err != nil {
		return nil, err
	}
	stored, err := p.findByPrefix(ctx, apiKey.Prefix)
	if err != nil {
		return nil, err
	}
	return &CreatedAPIKey{APIKey: stored, Key: rawKey}, nil
}

func (p *apiKeyPlugin) ListAPIKeys(ctx context.Context, userID any) ([]*APIKey, error) {
	models, err := p.core.FindMany(ctx, p.schema, []limen.Where{
		limen.Eq(p.schema.GetUserIDField(), userID),
	})
	if err != nil {
		return nil, err
	}
	keys := make([]*APIKey, 0, len(models))
	for _, model := range models {
		keys = append(keys, model.(*APIKey))
	}
	return keys, nil
}

func (p *apiKeyPlugin) RevokeAPIKey(ctx context.Context, userID any, id any) error {
	now := time.Now()
	return p.core.UpdateRaw(ctx, p.schema, &APIKey{
		RevokedAt: &now,
		UpdatedAt: now,
	}, []limen.Where{
		limen.Eq(p.schema.GetIDField(), id),
		limen.Eq(p.schema.GetUserIDField(), userID),
	}, true)
}

func (p *apiKeyPlugin) ValidateAPIKey(ctx context.Context, key string, scopes ...string) (*APIKey, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrAPIKeyRequired
	}

	apiKey, err := p.findByPrefix(ctx, p.storedPrefix(key))
	if errors.Is(err, limen.ErrRecordNotFound) {
		return nil, ErrAPIKeyInvalid
	}
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(apiKey.keyHash), []byte(p.hashKey(key))) != 1 {
		return nil, ErrAPIKeyInvalid
	}
	if apiKey.RevokedAt != nil {
		return nil, ErrAPIKeyRevoked
	}
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, ErrAPIKeyExpired
	}
	if !hasScopes(apiKey.Scopes, scopes) {
		return nil, ErrAPIKeyScope
	}

	now := time.Now()
	apiKey.LastUsedAt = &now
	apiKey.UpdatedAt = now
	if err := p.core.UpdateRaw(ctx, p.schema, apiKey, []limen.Where{
		limen.Eq(p.schema.GetIDField(), apiKey.ID),
	}, true); err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (p *apiKeyPlugin) findByPrefix(ctx context.Context, prefix string) (*APIKey, error) {
	model, err := p.core.FindOne(ctx, p.schema, []limen.Where{
		limen.Eq(p.schema.GetPrefixField(), prefix),
	}, nil)
	if err != nil {
		return nil, err
	}
	return model.(*APIKey), nil
}

func (p *apiKeyPlugin) generateKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return p.config.keyPrefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (p *apiKeyPlugin) storedPrefix(key string) string {
	if len(key) <= p.config.storedPrefixSize {
		return key
	}
	return key[:p.config.storedPrefixSize]
}

func (p *apiKeyPlugin) hashKey(key string) string {
	mac := hmac.New(sha256.New, p.core.Secret())
	_, _ = mac.Write([]byte(key))
	return hex.EncodeToString(mac.Sum(nil))
}

func (p *apiKeyPlugin) validateRequestedScopes(scopes []string) ([]string, error) {
	normalized, err := normalizeScopes(scopes)
	if err != nil {
		return nil, err
	}
	for _, scope := range normalized {
		if _, ok := p.config.allowedScopes[scope]; !ok {
			return nil, limen.NewLimenError("API key scope is not allowed", http.StatusForbidden, nil)
		}
	}
	return normalized, nil
}

func normalizeScopes(scopes []string) ([]string, error) {
	out := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		normalized, ok := normalizeScope(scope)
		if !ok {
			return nil, limen.NewLimenError("API key scope is invalid", http.StatusUnprocessableEntity, nil)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		out = append(out, normalized)
		seen[normalized] = struct{}{}
	}
	return out, nil
}

func normalizeScope(scope string) (string, bool) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "", false
	}
	for _, r := range scope {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == ':' || r == '.' || r == '_' || r == '-' || r == '/':
		default:
			return "", false
		}
	}
	return scope, true
}

func hasScopes(actual []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	if len(actual) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(actual))
	for _, scope := range actual {
		set[scope] = struct{}{}
	}
	for _, scope := range required {
		if _, ok := set[scope]; !ok {
			return false
		}
	}
	return true
}
