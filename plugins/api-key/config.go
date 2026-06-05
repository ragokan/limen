package apikey

import "time"

type config struct {
	headerName       string
	authorization    string
	keyPrefix        string
	storedPrefixSize int
	defaultTTL       time.Duration
	allowedScopes    map[string]struct{}
}

type ConfigOption func(*config)

func WithHeaderName(name string) ConfigOption {
	return func(c *config) {
		c.headerName = name
	}
}

func WithAuthorizationSchemePrefix(prefix string) ConfigOption {
	return func(c *config) {
		c.authorization = prefix
	}
}

func WithKeyPrefix(prefix string) ConfigOption {
	return func(c *config) {
		c.keyPrefix = prefix
	}
}

func WithStoredPrefixSize(size int) ConfigOption {
	return func(c *config) {
		c.storedPrefixSize = size
	}
}

func WithDefaultTTL(ttl time.Duration) ConfigOption {
	return func(c *config) {
		c.defaultTTL = ttl
	}
}

func WithAllowedScopes(scopes ...string) ConfigOption {
	return func(c *config) {
		if c.allowedScopes == nil {
			c.allowedScopes = make(map[string]struct{}, len(scopes))
		}
		for _, scope := range scopes {
			if normalized, ok := normalizeScope(scope); ok {
				c.allowedScopes[normalized] = struct{}{}
			}
		}
	}
}
