package apikey

import "time"

type APIKey struct {
	ID         any        `json:"id"`
	UserID     any        `json:"user_id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	keyHash    string
	raw        map[string]any
}

func (k APIKey) Raw() map[string]any {
	return k.raw
}

type CreatedAPIKey struct {
	APIKey *APIKey `json:"api_key"`
	Key    string  `json:"key"`
}

type CreateAPIKeyOptions struct {
	Scopes    []string
	ExpiresAt *time.Time
}

type CreateAPIKeyOption func(*CreateAPIKeyOptions)

func WithScopes(scopes ...string) CreateAPIKeyOption {
	return func(o *CreateAPIKeyOptions) {
		o.Scopes = append([]string(nil), scopes...)
	}
}

func WithExpiresAt(expiresAt time.Time) CreateAPIKeyOption {
	return func(o *CreateAPIKeyOptions) {
		o.ExpiresAt = &expiresAt
	}
}
