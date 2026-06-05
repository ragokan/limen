package admin

import (
	"net/http"

	"github.com/ragokan/limen"
)

var (
	ErrAdminNotConfigured = limen.NewLimenError("admin plugin has no configured administrators", http.StatusForbidden, nil)
	ErrAdminForbidden     = limen.NewLimenError("admin access is forbidden", http.StatusForbidden, nil)
)
