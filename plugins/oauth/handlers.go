package oauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ragokan/limen"
)

type oauthHandlers struct {
	plugin    *oauthPlugin
	responder *limen.Responder
}

func newOAuthHandlers(plugin *oauthPlugin, httpCore *limen.LimenHTTPCore) *oauthHandlers {
	return &oauthHandlers{
		plugin:    plugin,
		responder: httpCore.Responder,
	}
}

func (h *oauthHandlers) SignInWithOAuth(w http.ResponseWriter, r *http.Request) {
	providerName := limen.GetParam(r, "provider")

	request := &OAuthAuthorizeURLData{
		RedirectURI:      r.URL.Query().Get("redirect_uri"),
		ErrorRedirectURI: r.URL.Query().Get("error_redirect_uri"),
	}
	authURL, cookieValue, err := h.plugin.GetAuthorizationURL(r.Context(), providerName, request)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.setStateCookie(w, cookieValue)
	h.responder.JSON(w, r, http.StatusOK, map[string]any{"url": authURL})
}

func (h *oauthHandlers) Callback(w http.ResponseWriter, r *http.Request) {
	providerName := limen.GetParam(r, "provider")
	callbackParams, err := h.callbackParams(w, r)
	if err != nil {
		h.handleCallbackResponse(w, r, nil, nil, nil, err)
		return
	}
	code := callbackParams.Get(callbackParamCode)
	state := callbackParams.Get("state")
	callbackErr := callbackErrorFromQuery(callbackParams)

	cookieValue, err := h.plugin.core.Cookies().Get(r, h.plugin.config.cookieName)
	if err != nil {
		h.handleCallbackResponse(w, r, nil, nil, nil, ErrMissingStateCookie)
		return
	}

	h.clearStateCookie(w)

	ctx := ContextWithCallbackParams(r.Context(), callbackParams)
	result, stateData, err := h.plugin.AuthenticateWithProvider(ctx, providerName, code, state, cookieValue, callbackErr)
	if err != nil {
		h.handleCallbackResponse(w, r, stateData, nil, nil, err)
		return
	}

	var sessionResult *limen.SessionResult
	if stateData[linkUserIdKey] == nil {
		sessionResult, err = h.plugin.core.CreateSession(r.Context(), r, w, result)
		if err != nil {
			h.handleCallbackResponse(w, r, stateData, nil, nil, err)
			return
		}
	}

	h.handleCallbackResponse(w, r, stateData, result, sessionResult, err)
}

// LinkAccountWithOAuth initiates the OAuth flow for linking a provider to the current user's account.
func (h *oauthHandlers) LinkAccountWithOAuth(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	providerName := limen.GetParam(r, "provider")
	data := map[string]any{
		linkUserIdKey: session.User.ID,
	}
	request := &OAuthAuthorizeURLData{
		AdditionalData:   data,
		RedirectURI:      r.URL.Query().Get("redirect_uri"),
		ErrorRedirectURI: r.URL.Query().Get("error_redirect_uri"),
	}

	authURL, cookieValue, err := h.plugin.GetAuthorizationURL(r.Context(), providerName, request)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.setStateCookie(w, cookieValue)
	h.responder.JSON(w, r, http.StatusOK, map[string]any{"url": authURL})
}

func (h *oauthHandlers) ListAccounts(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	accounts, err := h.plugin.ListAccountsForUser(r.Context(), session.User.ID)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, limen.SerializeAll(h.plugin.accountSchema, accounts))
}

func (h *oauthHandlers) UnlinkAccount(w http.ResponseWriter, r *http.Request) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	providerName := limen.GetParam(r, "provider")

	err = h.plugin.UnlinkAccount(r.Context(), session.User, providerName)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusNoContent, nil)
}

func (h *oauthHandlers) GetTokens(w http.ResponseWriter, r *http.Request) {
	h.respondWithTokens(w, r, h.plugin.GetAccessToken)
}

func (h *oauthHandlers) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	h.respondWithTokens(w, r, h.plugin.RefreshAccessToken)
}

func (h *oauthHandlers) respondWithTokens(w http.ResponseWriter, r *http.Request, getTokens func(context.Context, any, string) (*ActiveTokens, error)) {
	session, err := limen.GetCurrentSessionFromCtx(r)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	providerName := limen.GetParam(r, "provider")

	tokens, err := getTokens(r.Context(), session.User.ID, providerName)
	if err != nil {
		h.responder.Error(w, r, err)
		return
	}

	h.responder.JSON(w, r, http.StatusOK, tokens)
}

func (h *oauthHandlers) handleCallbackResponse(w http.ResponseWriter, r *http.Request, stateData map[string]any, authResult *limen.AuthenticationResult, sessionResult *limen.SessionResult, err error) {
	if (h.plugin.config.disableRedirect && err != nil) || stateData == nil {
		h.responder.Error(w, r, err)
		return
	}

	if h.plugin.config.disableRedirect {
		h.responder.SessionResponse(w, r, h.plugin.core, authResult, sessionResult)
		return
	}

	redirectURI, _ := stateData[redirectURIKey].(string)
	errorRedirectURI, _ := stateData[errorRedirectURIKey].(string)
	if err != nil && errorRedirectURI != "" {
		redirectURI = errorRedirectURI
	}

	if err != nil {
		redirectURI = h.buildErrorRedirectURL(redirectURI, err)
	}

	h.responder.RedirectWithSession(w, r, redirectURI, sessionResult)
}

// buildErrorRedirectURL appends error query parameters to the redirect URL.
// When the error carries structured OAuth details (code, error_description),
// those are forwarded as separate params per RFC 6749. Otherwise the error
// message is placed in a single "error" param.
func (h *oauthHandlers) buildErrorRedirectURL(redirectURI string, err error) string {
	ae := limen.ToLimenError(err)
	if details, ok := ae.Details().(map[string]string); ok {
		code := details[callbackParamCode]
		if code != "" {
			return appendOAuthErrorParams(redirectURI, code, details[callbackParamErrorDescription])
		}
	}

	return appendOAuthErrorParams(redirectURI, ae.Error(), "")
}

// FormPostCallback handles OAuth callbacks delivered via response_mode=form_post.
// The IdP POSTs code/state/error as application/x-www-form-urlencoded. Cross-site
// POST callbacks may not include SameSite=Lax cookies, so we store the POST body
// in an encrypted short-lived cookie and redirect to the GET callback with only a
// marker query parameter. The browser follows that same-site GET with cookies.
func (h *oauthHandlers) FormPostCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.responder.Error(w, r, err)
		return
	}

	params := r.URL.Query()
	for key, values := range r.PostForm {
		params.Del(key)
		for _, value := range values {
			params.Add(key, value)
		}
	}

	encrypted, err := limen.EncryptXChaCha(params.Encode(), h.plugin.config.secret, nil)
	if err != nil {
		h.responder.Error(w, r, fmt.Errorf("oauth: failed to store form_post callback params: %w", err))
		return
	}
	h.plugin.core.Cookies().Set(w, formPostCookieName, encrypted, 60)

	target := url.URL{
		Path: r.URL.Path,
	}
	query := target.Query()
	query.Set(formPostQueryKey, "1")
	target.RawQuery = query.Encode()
	h.responder.Redirect(w, r, target.String(), http.StatusSeeOther)
}

func (h *oauthHandlers) callbackParams(w http.ResponseWriter, r *http.Request) (url.Values, error) {
	if r.URL.Query().Get(formPostQueryKey) != "1" {
		return r.URL.Query(), nil
	}

	cookieValue, err := h.plugin.core.Cookies().Get(r, formPostCookieName)
	if err != nil {
		return nil, limen.NewLimenError("missing OAuth form_post callback cookie", http.StatusBadRequest, err)
	}
	h.plugin.core.Cookies().Delete(w, formPostCookieName)

	raw, err := limen.DecryptXChaCha(cookieValue, h.plugin.config.secret, nil)
	if err != nil {
		return nil, limen.NewLimenError("invalid OAuth form_post callback cookie", http.StatusBadRequest, err)
	}
	params, err := url.ParseQuery(raw)
	if err != nil {
		return nil, limen.NewLimenError("invalid OAuth form_post callback params", http.StatusBadRequest, err)
	}
	return params, nil
}

func (h *oauthHandlers) setStateCookie(w http.ResponseWriter, value string) {
	h.plugin.core.Cookies().Set(w, h.plugin.config.cookieName, value, int(h.plugin.config.cookieTTL.Seconds()))
}

func (h *oauthHandlers) clearStateCookie(w http.ResponseWriter) {
	h.plugin.core.Cookies().Delete(w, h.plugin.config.cookieName)
}
