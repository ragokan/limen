package admin

import (
	"net/http"

	"github.com/ragokan/limen"
)

type handlers struct {
	plugin    *adminPlugin
	responder *limen.Responder
}

func newHandlers(plugin *adminPlugin, httpCore *limen.LimenHTTPCore) *handlers {
	return &handlers{
		plugin:    plugin,
		responder: httpCore.Responder,
	}
}

func (h *handlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.plugin.ListUsers(r.Context())
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, users)
}

func (h *handlers) GetUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.plugin.GetUser(r.Context(), parseID(limen.GetParam(r, "id")))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, user)
}

func (h *handlers) RevokeUserSessions(w http.ResponseWriter, r *http.Request) {
	if err := h.plugin.RevokeUserSessions(r.Context(), parseID(limen.GetParam(r, "id"))); err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusNoContent, nil)
}
