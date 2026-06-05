package organization

import (
	"net/http"

	"github.com/ragokan/limen"
)

type handlers struct {
	plugin    *organizationPlugin
	responder *limen.Responder
}

func newHandlers(plugin *organizationPlugin, httpCore *limen.LimenHTTPCore) *handlers {
	return &handlers{plugin: plugin, responder: httpCore.Responder}
}

func (h *handlers) Create(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("name", data["name"]).RequiredString("slug", data["slug"])
	})
	if body == nil {
		return
	}
	org, err := h.plugin.CreateOrganization(r.Context(), session.User.ID, body["name"].(string), body["slug"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusCreated, org)
}

func (h *handlers) List(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	orgs, err := h.plugin.ListOrganizationsForUser(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, orgs)
}

func (h *handlers) AddMember(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("role", data["role"])
	})
	if body == nil {
		return
	}
	userID, ok := body["user_id"]
	if !ok {
		h.responder.Error(w, r, limen.NewLimenError("user_id is required", http.StatusUnprocessableEntity, nil))
		return
	}
	membership, err := h.plugin.AddMember(r.Context(), parseID(limen.GetParam(r, "id")), normalizeBodyID(userID), Role(body["role"].(string)))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusCreated, membership)
}

func (h *handlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	err := h.plugin.RemoveMember(r.Context(), parseID(limen.GetParam(r, "id")), parseID(limen.GetParam(r, "user_id")))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *handlers) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("email", data["email"]).Email("email", data["email"]).RequiredString("role", data["role"])
	})
	if body == nil {
		return
	}
	invitation, err := h.plugin.CreateInvitation(r.Context(), parseID(limen.GetParam(r, "id")), body["email"].(string), Role(body["role"].(string)))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusCreated, invitation)
}

func (h *handlers) ListInvitations(w http.ResponseWriter, r *http.Request) {
	invitations, err := h.plugin.ListInvitations(r.Context(), parseID(limen.GetParam(r, "id")))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, invitations)
}

func (h *handlers) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	body := limen.ValidateJSON(w, r, h.responder, func(v *limen.Validator, data map[string]any) *limen.Validator {
		return v.RequiredString("token", data["token"])
	})
	if body == nil {
		return
	}
	membership, err := h.plugin.AcceptInvitation(r.Context(), session.User.ID, body["token"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}
	h.responder.JSON(w, r, http.StatusOK, membership)
}

func normalizeBodyID(value any) any {
	if number, ok := value.(float64); ok {
		return int64(number)
	}
	return value
}
