package httpserver

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/chiririll/savvy-plus/internal/domain"
	"github.com/go-chi/chi/v5"
)

func (s *Server) ssoProviders(w http.ResponseWriter, r *http.Request) {
	list, err := s.sso.Enabled(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, p := range list {
		data = append(data, p.PublicJSON())
	}
	writeData(w, http.StatusOK, data)
}

func (s *Server) ssoPresets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, domain.SSOPresetCatalog())
}

func (s *Server) ssoRedirect(w http.ResponseWriter, r *http.Request) {
	p := s.lookupProvider(r, chi.URLParam(r, "slug"))
	if p == nil || !p.Enabled {
		s.redirectLogin(w, r, "sso_error")
		return
	}
	loc, err := s.sso.AuthorizeURL(r.Context(), *p, r.URL.Query().Get("redirect"))
	if err != nil {
		s.redirectLogin(w, r, ssoCode(err))
		return
	}
	http.Redirect(w, r, loc, http.StatusFound)
}

func (s *Server) ssoCallback(w http.ResponseWriter, r *http.Request) {
	p := s.lookupProvider(r, chi.URLParam(r, "slug"))
	if p == nil || !p.Enabled {
		s.redirectSPAError(w, r, "sso_error")
		return
	}
	id, err := s.sso.HandleCallback(r.Context(), *p, r.URL.Query())
	s.finishSSO(w, r, p, id, err)
}

func (s *Server) ssoACS(w http.ResponseWriter, r *http.Request) {
	p := s.lookupProvider(r, chi.URLParam(r, "slug"))
	if p == nil || !p.Enabled {
		s.redirectSPAError(w, r, "sso_error")
		return
	}
	_ = r.ParseForm()
	id, err := s.sso.HandleACS(r.Context(), *p, r.FormValue("SAMLResponse"), r.FormValue("RelayState"))
	s.finishSSO(w, r, p, id, err)
}

func (s *Server) ssoMetadata(w http.ResponseWriter, r *http.Request) {
	p, err := s.sso.BySlug(r.Context(), chi.URLParam(r, "slug"))
	if err != nil || p == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(s.sso.SamlMetadata(*p)))
}

func (s *Server) ssoExchange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ticket string `json:"ticket"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Ticket == "" {
		writeValidation(w, map[string][]string{"ticket": {"The given data was invalid."}})
		return
	}
	userID, requires2FA, ok := s.sso.ConsumeTicket(r.Context(), body.Ticket)
	if !ok {
		writeJSON(w, http.StatusGone, map[string]any{"message": "Login ticket is invalid or expired.", "error": "invalid_ticket"})
		return
	}
	u, err := s.users.ByID(r.Context(), userID)
	if err != nil || u == nil {
		writeJSON(w, http.StatusGone, map[string]any{"message": "Login ticket is invalid or expired.", "error": "invalid_ticket"})
		return
	}
	if requires2FA {
		tok, err := s.challenges.Issue(r.Context(), u)
		if err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"requires_2fa": true, "two_factor_token": tok})
		return
	}
	s.issueSession(w, r, u, http.StatusOK, true)
}

func (s *Server) finishSSO(w http.ResponseWriter, r *http.Request, p *domain.IdentityProvider, id domain.NormalizedIdentity, err error) {
	if err != nil {
		s.redirectSPAError(w, r, ssoCode(err))
		return
	}
	user, needs2FA, err := s.sso.Provision(r.Context(), *p, id)
	if err != nil {
		s.redirectSPAError(w, r, ssoCode(err))
		return
	}
	ticket, err := s.sso.IssueTicket(r.Context(), user.ID, needs2FA)
	if err != nil {
		s.redirectSPAError(w, r, "sso_error")
		return
	}
	http.Redirect(w, r, strings.TrimRight(s.cfg.AppURL, "/")+"/auth/sso/callback?ticket="+ticket, http.StatusFound)
}

func (s *Server) lookupProvider(r *http.Request, slug string) *domain.IdentityProvider {
	p, err := s.sso.BySlug(r.Context(), slug)
	if err != nil {
		return nil
	}
	return p
}

func (s *Server) redirectLogin(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, strings.TrimRight(s.cfg.AppURL, "/")+"/login?sso_error="+code, http.StatusFound)
}

func (s *Server) redirectSPAError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, strings.TrimRight(s.cfg.AppURL, "/")+"/auth/sso/callback?error="+code, http.StatusFound)
}

func ssoCode(err error) string {
	if e := domain.AsSSOError(err); e != nil {
		return e.Code
	}
	return "sso_error"
}

func (s *Server) idpIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.sso.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, p := range list {
		data = append(data, p.JSON(s.cfg.AppURL))
	}
	writeData(w, http.StatusOK, data)
}

func (s *Server) idpShow(w http.ResponseWriter, r *http.Request) {
	p := s.idpParam(w, r)
	if p == nil {
		return
	}
	writeJSON(w, http.StatusOK, p.JSON(s.cfg.AppURL))
}

func (s *Server) idpStore(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeIdPWrite(w, r, true)
	if !ok {
		return
	}
	p, err := s.sso.Create(r.Context(), in)
	if err != nil {
		writeSSOErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p.JSON(s.cfg.AppURL))
}

func (s *Server) idpUpdate(w http.ResponseWriter, r *http.Request) {
	p := s.idpParam(w, r)
	if p == nil {
		return
	}
	in, ok := decodeIdPWrite(w, r, false)
	if !ok {
		return
	}
	updated, err := s.sso.Update(r.Context(), p.ID, in)
	if err != nil {
		writeSSOErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated.JSON(s.cfg.AppURL))
}

func (s *Server) idpDestroy(w http.ResponseWriter, r *http.Request) {
	p := s.idpParam(w, r)
	if p == nil {
		return
	}
	if err := s.sso.Delete(r.Context(), p.ID); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"message": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) idpTest(w http.ResponseWriter, r *http.Request) {
	p := s.idpParam(w, r)
	if p == nil {
		return
	}
	if err := s.sso.TestConnection(r.Context(), *p); err != nil {
		code := "sso_error"
		status := http.StatusBadGateway
		msg := err.Error()
		if e := domain.AsSSOError(err); e != nil {
			code, status, msg = e.Code, e.Status, e.Message
		}
		writeJSON(w, status, map[string]any{"status": "error", "message": msg, "error": code})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) idpParam(w http.ResponseWriter, r *http.Request) *domain.IdentityProvider {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	p, err := s.sso.ByID(r.Context(), id)
	if err != nil || p == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	return p
}

func decodeIdPWrite(w http.ResponseWriter, r *http.Request, create bool) (domain.IdPWrite, bool) {
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return domain.IdPWrite{}, false
	}
	var in domain.IdPWrite
	in.Name, _ = raw["name"].(string)
	in.Slug, _ = raw["slug"].(string)
	in.Preset, _ = raw["preset"].(string)
	in.DefaultRole, _ = raw["default_role"].(string)
	if v, ok := raw["enabled"]; ok {
		b := asBool(v)
		in.Enabled = &b
	}
	if v, ok := raw["allow_jit"]; ok {
		b := asBool(v)
		in.AllowJIT = &b
	}
	if v, ok := raw["sync_role_on_login"]; ok {
		b := asBool(v)
		in.SyncRoleOnLogin = &b
	}
	if v, ok := raw["link_by_email"]; ok {
		b := asBool(v)
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
			writeValidation(w, errs)
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
	if e := domain.AsSSOError(err); e != nil {
		if e.Status == 422 && e.Code == "invalid_slug" {
			writeValidation(w, map[string][]string{"slug": {e.Message}})
			return
		}
		if e.Status == 422 && e.Code == "slug_taken" {
			writeValidation(w, map[string][]string{"slug": {e.Message}})
			return
		}
		writeJSON(w, e.Status, map[string]any{"message": e.Message, "error": e.Code})
		return
	}
	writeMessage(w, http.StatusInternalServerError, err.Error())
}
