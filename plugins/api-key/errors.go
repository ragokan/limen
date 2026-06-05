package apikey

import (
	"net/http"

	"github.com/ragokan/limen"
)

var (
	ErrAPIKeyInvalid  = limen.NewLimenError("API key is invalid", http.StatusUnauthorized, nil)
	ErrAPIKeyExpired  = limen.NewLimenError("API key has expired", http.StatusUnauthorized, nil)
	ErrAPIKeyRevoked  = limen.NewLimenError("API key has been revoked", http.StatusUnauthorized, nil)
	ErrAPIKeyScope    = limen.NewLimenError("API key scope is not allowed", http.StatusForbidden, nil)
	ErrAPIKeyRequired = limen.NewLimenError("API key is required", http.StatusUnauthorized, nil)
)
