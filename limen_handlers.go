package limen

import (
	"net/http"
	"time"
)

type limenHandlers struct {
	core      *LimenCore
	responder *Responder
	config    *httpConfig
}

type sessionListItem struct {
	ID         any       `json:"id,omitempty"`
	UserID     any       `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastAccess time.Time `json:"last_access"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
}

func registerBaseRoutes(router *router, httpCore *LimenHTTPCore, core *LimenCore, basePath string) {
	routeBuilder := &RouteBuilder{
		group: router.Group(basePath),
		core:  httpCore,
	}
	handlers := newLimenHandlers(httpCore, core)
	handlers.RegisterRoutes(routeBuilder)
}

func newLimenHandlers(httpCore *LimenHTTPCore, core *LimenCore) *limenHandlers {
	return &limenHandlers{
		core:      core,
		responder: httpCore.Responder,
		config:    httpCore.config,
	}
}

func (h *limenHandlers) RegisterRoutes(routeBuilder *RouteBuilder) {
	routeBuilder.ProtectedGETWithMetadata("/me", "me", h.GetSession, coreRouteMetadata("Get current session"))
	routeBuilder.ProtectedGETWithMetadata("/sessions", "list-sessions", h.ListSessions, coreRouteMetadata("List sessions"))
	routeBuilder.ProtectedPOSTWithMetadata("/signout", "signout", h.SignOut, coreRouteMetadata(
		"Sign out",
		WithRouteResponse(http.StatusNoContent, OpenAPIResponse{Description: "No Content"}),
	))
	routeBuilder.ProtectedPOSTWithMetadata("/revoke-sessions", "revoke-sessions", h.RevokeAllSessions, coreRouteMetadata(
		"Revoke sessions",
		WithRouteResponse(http.StatusNoContent, OpenAPIResponse{Description: "No Content"}),
	))

	if h.core.EmailVerificationEnabled() {
		routeBuilder.POSTWithMetadata("/verify-email", "verify-email", h.VerifyEmail, coreRouteMetadata(
			"Verify email",
			WithRouteAllowedContentTypes("application/json"),
		))
		routeBuilder.ProtectedPOSTWithMetadata("/email-verifications", "email-verifications", h.RequestEmailVerification, coreRouteMetadata(
			"Request email verification",
			WithRouteAllowedContentTypes("application/json"),
		))
	}
}

func coreRouteMetadata(summary string, opts ...RouteMetadataOption) *RouteMetadata {
	options := []RouteMetadataOption{
		WithRouteSummary(summary),
		WithRouteTags("auth"),
	}
	options = append(options, opts...)
	return NewRouteMetadata(options...)
}

func (h *limenHandlers) GetSession(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.core.Cookies().ClearSessionCookie(w)
		h.responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	h.responder.SessionResponse(w, r, h.core, &AuthenticationResult{User: session.User}, nil)
}

func (h *limenHandlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	sessions, err := h.core.SessionManager.ListSessions(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, redactedSessionList(sessions))
}

func redactedSessionList(sessions []Session) []sessionListItem {
	out := make([]sessionListItem, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, sessionListItem{
			ID:         session.ID,
			UserID:     session.UserID,
			CreatedAt:  session.CreatedAt,
			ExpiresAt:  session.ExpiresAt,
			LastAccess: session.LastAccess,
			IPAddress:  sessionMetadataString(session.Metadata, "ip_address"),
			UserAgent:  sessionMetadataString(session.Metadata, "user_agent"),
		})
	}
	return out
}

func sessionMetadataString(metadata map[string]any, key string) string {
	value, _ := metadata[key].(string)
	return value
}

func (h *limenHandlers) RevokeAllSessions(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	err = h.core.SessionManager.RevokeAllSessions(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *limenHandlers) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	body := ValidateJSON(w, r, h.responder, func(v *Validator, data map[string]any) *Validator {
		return v.RequiredString("token", data["token"])
	})
	if body == nil {
		return
	}

	err := h.core.VerifyEmail(r.Context(), body["token"].(string))
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, "email verified successfully")
}

func (h *limenHandlers) RequestEmailVerification(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	_, err = h.core.RequestEmailVerification(r.Context(), &User{
		Email: session.User.Email,
	}, true)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, "email verification requested successfully")
}

func (h *limenHandlers) SignOut(w http.ResponseWriter, r *http.Request) {
	session, err := GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, NewLimenError(err.Error(), http.StatusUnauthorized, nil))
		return
	}

	err = h.core.SessionManager.RevokeSession(r.Context(), session.Session.Token)
	if err != nil {
		h.responder.Error(w, r, NewLimenError(err.Error(), http.StatusBadRequest, nil))
		return
	}

	h.core.Cookies().ClearSessionCookie(w)

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}
