// Package sso serves single sign-on.
package sso

import (
	"encoding/json"
	"net/http"
	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/guard"
	"savvy-go/internal/httpserver/httpx/login"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/sso"
)

// Handler holds what the sign-in and provider routes need.
type Handler struct {
	Cfg        config.Config
	SSO        *sso.Service
	Users      auth.Users
	Challenges auth.Challenges
	Login      login.Service
	Guard      *guard.Guard
}

func (h Handler) ssoProviders(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.URL() == "" {
		httpx.WriteData(w, http.StatusOK, []any{})
		return
	}
	list, err := h.SSO.Enabled(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, p := range list {
		data = append(data, p.PublicJSON())
	}
	httpx.WriteData(w, http.StatusOK, data)
}

func (h Handler) ssoRedirect(w http.ResponseWriter, r *http.Request) {
	p := h.lookupProvider(r, chi.URLParam(r, "slug"))
	if p == nil || !p.Enabled {
		h.redirectLogin(w, r, "sso_error")
		return
	}
	loc, err := h.SSO.AuthorizeURL(r.Context(), *p, r.URL.Query().Get("redirect"))
	if err != nil {
		h.redirectLogin(w, r, ssoCode(err))
		return
	}
	http.Redirect(w, r, loc, http.StatusFound)
}

func (h Handler) ssoCallback(w http.ResponseWriter, r *http.Request) {
	p := h.lookupProvider(r, chi.URLParam(r, "slug"))
	if p == nil || !p.Enabled {
		h.redirectSPAError(w, r, "sso_error")
		return
	}
	id, err := h.SSO.HandleCallback(r.Context(), *p, r.URL.Query())
	h.finishSSO(w, r, p, id, err)
}

func (h Handler) ssoACS(w http.ResponseWriter, r *http.Request) {
	p := h.lookupProvider(r, chi.URLParam(r, "slug"))
	if p == nil || !p.Enabled {
		h.redirectSPAError(w, r, "sso_error")
		return
	}
	_ = r.ParseForm()
	id, err := h.SSO.HandleACS(r.Context(), *p, r.FormValue("SAMLResponse"), r.FormValue("RelayState"))
	h.finishSSO(w, r, p, id, err)
}

func (h Handler) ssoMetadata(w http.ResponseWriter, r *http.Request) {
	p, err := h.SSO.BySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil || p == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(h.SSO.SamlMetadata(r.Context(), *p)))
}

func (h Handler) ssoExchange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ticket string `json:"ticket"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Ticket == "" {
		httpx.WriteValidation(w, map[string][]string{"ticket": {"The given data was invalid."}})
		return
	}
	userID, requires2FA, ok := h.SSO.ConsumeTicket(r.Context(), body.Ticket)
	if !ok {
		httpx.WriteCoded(w, http.StatusGone, "Login ticket is invalid or expired.", "invalid_ticket")
		return
	}
	u, err := h.Users.ByID(r.Context(), userID)
	if err != nil || u == nil {
		httpx.WriteCoded(w, http.StatusGone, "Login ticket is invalid or expired.", "invalid_ticket")
		return
	}
	if requires2FA {
		tok, err := h.Challenges.Issue(r.Context(), u)
		if err != nil {
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"requires_2fa": true, "two_factor_token": tok})
		return
	}
	h.Login.Issue(w, r, u, http.StatusOK, true)
}

func (h Handler) finishSSO(w http.ResponseWriter, r *http.Request, p *sso.IdentityProvider, id sso.NormalizedIdentity, err error) {
	if err != nil {
		h.redirectSPAError(w, r, ssoCode(err))
		return
	}
	user, needs2FA, err := h.SSO.Provision(r.Context(), *p, id)
	if err != nil {
		h.redirectSPAError(w, r, ssoCode(err))
		return
	}
	ticket, err := h.SSO.IssueTicket(r.Context(), user.ID, needs2FA)
	if err != nil {
		h.redirectSPAError(w, r, "sso_error")
		return
	}
	http.Redirect(w, r, "/auth/sso/callback?ticket="+ticket, http.StatusFound)
}

func (h Handler) lookupProvider(r *http.Request, slug string) *sso.IdentityProvider {
	p, err := h.SSO.BySlug(r.Context(), slug)
	if err != nil {
		return nil
	}
	return p
}

func (h Handler) redirectLogin(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/login?sso_error="+code, http.StatusFound)
}

func (h Handler) redirectSPAError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, "/auth/sso/callback?error="+code, http.StatusFound)
}

func ssoCode(err error) string {
	if e := sso.AsError(err); e != nil {
		return e.Code
	}
	return "sso_error"
}
