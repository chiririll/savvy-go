// Package ssoapi serves single sign-on and the identity providers an admin sets up.
package ssoapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"savvy-go/internal/auth"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/guard"
	"savvy-go/internal/httpserver/signin"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/sso"
)

// Handler holds what the sign-in and provider routes need.
type Handler struct {
	Cfg        config.Config
	SSO        *sso.Service
	Users      auth.Users
	Challenges auth.Challenges
	Login      signin.Service
	Guard      *guard.Guard
}

func (h Handler) ssoProviders(w http.ResponseWriter, r *http.Request) {
	if h.Cfg.URL() == "" {
		api.WriteData(w, http.StatusOK, []any{})
		return
	}
	list, err := h.SSO.Enabled(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, p := range list {
		data = append(data, p.PublicJSON())
	}
	api.WriteData(w, http.StatusOK, data)
}

func (h Handler) ssoPresets(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, sso.PresetCatalog())
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
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
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
		api.WriteValidation(w, map[string][]string{"ticket": {"The given data was invalid."}})
		return
	}
	userID, requires2FA, ok := h.SSO.ConsumeTicket(r.Context(), body.Ticket)
	if !ok {
		api.WriteJSON(w, http.StatusGone, map[string]any{"message": "Login ticket is invalid or expired.", "error": "invalid_ticket"})
		return
	}
	u, err := h.Users.ByID(r.Context(), userID)
	if err != nil || u == nil {
		api.WriteJSON(w, http.StatusGone, map[string]any{"message": "Login ticket is invalid or expired.", "error": "invalid_ticket"})
		return
	}
	if requires2FA {
		tok, err := h.Challenges.Issue(r.Context(), u)
		if err != nil {
			api.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		api.WriteJSON(w, http.StatusOK, map[string]any{"requires_2fa": true, "two_factor_token": tok})
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

func (h Handler) idpIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.SSO.All(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, p := range list {
		data = append(data, p.JSON(h.Cfg.URL()))
	}
	api.WriteData(w, http.StatusOK, data)
}

func (h Handler) idpShow(w http.ResponseWriter, r *http.Request) {
	p := h.idpParam(w, r)
	if p == nil {
		return
	}
	api.WriteJSON(w, http.StatusOK, p.JSON(h.Cfg.URL()))
}

func (h Handler) idpStore(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeIdPWrite(w, r, true)
	if !ok {
		return
	}
	p, err := h.SSO.Create(r.Context(), in)
	if err != nil {
		writeSSOErr(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, p.JSON(h.Cfg.URL()))
}

func (h Handler) idpUpdate(w http.ResponseWriter, r *http.Request) {
	p := h.idpParam(w, r)
	if p == nil {
		return
	}
	in, ok := decodeIdPWrite(w, r, false)
	if !ok {
		return
	}
	updated, err := h.SSO.Update(r.Context(), p.ID, in)
	if err != nil {
		writeSSOErr(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, updated.JSON(h.Cfg.URL()))
}

func (h Handler) idpDestroy(w http.ResponseWriter, r *http.Request) {
	p := h.idpParam(w, r)
	if p == nil {
		return
	}
	if err := h.SSO.Delete(r.Context(), p.ID); err != nil {
		api.WriteJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) idpTest(w http.ResponseWriter, r *http.Request) {
	p := h.idpParam(w, r)
	if p == nil {
		return
	}
	if err := h.SSO.TestConnection(r.Context(), *p); err != nil {
		code := "sso_error"
		status := http.StatusBadGateway
		msg := err.Error()
		if e := sso.AsError(err); e != nil {
			code, status, msg = e.Code, e.Status, e.Message
		}
		api.WriteJSON(w, status, map[string]any{"status": "error", "message": msg, "error": code})
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h Handler) idpParam(w http.ResponseWriter, r *http.Request) *sso.IdentityProvider {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	p, err := h.SSO.ByID(r.Context(), id)
	if err != nil || p == nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	return p
}

func decodeIdPWrite(w http.ResponseWriter, r *http.Request, create bool) (sso.IdPWrite, bool) {
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		api.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return sso.IdPWrite{}, false
	}
	var in sso.IdPWrite
	in.Name, _ = raw["name"].(string)
	in.Slug, _ = raw["slug"].(string)
	in.Preset, _ = raw["preset"].(string)
	in.DefaultRole, _ = raw["default_role"].(string)
	if v, ok := raw["enabled"]; ok {
		b := api.AsBool(v)
		in.Enabled = &b
	}
	if v, ok := raw["allow_jit"]; ok {
		b := api.AsBool(v)
		in.AllowJIT = &b
	}
	if v, ok := raw["sync_role_on_login"]; ok {
		b := api.AsBool(v)
		in.SyncRoleOnLogin = &b
	}
	if v, ok := raw["link_by_email"]; ok {
		b := api.AsBool(v)
		in.LinkByEmail = &b
	}
	if v, ok := raw["sort_order"]; ok {
		n := int(asFloat(v))
		in.SortOrder = &n
	}
	if v, ok := raw["fields"].(map[string]any); ok {
		in.Fields = v
	}
	if v, ok := raw["claim_mappings"].(map[string]any); ok {
		in.ClaimMappings = v
	}
	if v, ok := raw["role_mapping"]; ok {
		in.RoleMapping = v
	}
	if create {
		errs := map[string][]string{}
		if strings.TrimSpace(in.Name) == "" {
			errs["name"] = []string{"The name field is required."}
		}
		if !idpSlugRE.MatchString(in.Slug) {
			errs["slug"] = []string{"The slug format is invalid."}
		}
		if in.Preset == "" {
			errs["preset"] = []string{"The preset field is required."}
		}
		if len(errs) > 0 {
			api.WriteValidation(w, errs)
			return in, false
		}
	}
	return in, true
}

var idpSlugRE = regexp.MustCompile(`^[a-z0-9-]+$`)

func asFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	default:
		return 0
	}
}

func writeSSOErr(w http.ResponseWriter, err error) {
	if e := sso.AsError(err); e != nil {
		if e.Status == 422 && e.Code == "invalid_slug" {
			api.WriteValidation(w, map[string][]string{"slug": {e.Message}})
			return
		}
		if e.Status == 422 && e.Code == "slug_taken" {
			api.WriteValidation(w, map[string][]string{"slug": {e.Message}})
			return
		}
		api.WriteJSON(w, e.Status, map[string]any{"message": e.Message, "error": e.Code})
		return
	}
	api.WriteMessage(w, http.StatusInternalServerError, err.Error())
}
