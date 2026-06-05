package apikey

import (
	"context"
	"net/http"
	"strings"

	"github.com/ragokan/limen"
)

type contextKey struct{}

func (p *apiKeyPlugin) MiddlewareRequireAPIKey(scopes ...string) limen.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := p.keyFromRequest(r)
			apiKey, err := p.ValidateAPIKey(r.Context(), key, scopes...)
			if err != nil {
				limenErr := limen.ToLimenError(err)
				http.Error(w, limenErr.Error(), limenErr.Status())
				return
			}
			ctx := context.WithValue(r.Context(), contextKey{}, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetAPIKeyFromContext(ctx context.Context) (*APIKey, bool) {
	key, ok := ctx.Value(contextKey{}).(*APIKey)
	return key, ok
}

func (p *apiKeyPlugin) keyFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if key := strings.TrimSpace(r.Header.Get(p.config.headerName)); key != "" {
		return key
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(auth, p.config.authorization) {
		return strings.TrimSpace(strings.TrimPrefix(auth, p.config.authorization))
	}
	return ""
}
