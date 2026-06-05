package apikey

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ragokan/limen"
)

type handlers struct {
	plugin    *apiKeyPlugin
	responder *limen.Responder
}

func newHandlers(plugin *apiKeyPlugin, httpCore *limen.LimenHTTPCore) *handlers {
	return &handlers{
		plugin:    plugin,
		responder: httpCore.Responder,
	}
}

func (h *handlers) Create(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("name", data["name"])
	})
	if body == nil {
		return
	}

	scopes, err := scopesFromBody(body["scopes"])
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	expiresAt, err := expiresAtFromBody(body["expires_at"])
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	var opts []CreateAPIKeyOption
	if len(scopes) > 0 {
		opts = append(opts, WithScopes(scopes...))
	}
	if expiresAt != nil {
		opts = append(opts, WithExpiresAt(*expiresAt))
	}

	created, err := h.plugin.CreateAPIKey(r.Context(), session.User.ID, body["name"].(string), opts...)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusCreated, created)
}

func (h *handlers) List(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	keys, err := h.plugin.ListAPIKeys(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, keys)
}

func (h *handlers) Revoke(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	id := parseID(limen.GetParam(r, "id"))
	if err := h.plugin.RevokeAPIKey(r.Context(), session.User.ID, id); err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func scopesFromBody(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	raw, ok := value.([]any)
	if !ok {
		return nil, limen.NewLimenError("scopes must be an array of strings", http.StatusUnprocessableEntity, nil)
	}
	scopes := make([]string, 0, len(raw))
	for _, item := range raw {
		scope, ok := item.(string)
		if !ok {
			return nil, limen.NewLimenError("scopes must be an array of strings", http.StatusUnprocessableEntity, nil)
		}
		scopes = append(scopes, scope)
	}
	return scopes, nil
}

func expiresAtFromBody(value any) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	raw, ok := value.(string)
	if !ok {
		return nil, limen.NewLimenError("expires_at must be an RFC3339 string", http.StatusUnprocessableEntity, nil)
	}
	expiresAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, limen.NewLimenError("expires_at must be an RFC3339 string", http.StatusUnprocessableEntity, err)
	}
	return &expiresAt, nil
}

func parseID(value string) any {
	if id, err := strconv.ParseInt(value, 10, 64); err == nil {
		return id
	}
	return value
}
