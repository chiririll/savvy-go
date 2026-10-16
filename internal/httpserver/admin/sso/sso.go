// Package sso serves the identity providers a server admin sets up for single sign-on.
package sso

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/config/shared"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/sso"
)

// Handler holds what the identity provider routes need.
type Handler struct {
	Cfg shared.Config
	SSO *sso.Service
}

func (h Handler) ssoPresets(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, sso.PresetCatalog())
}

func (h Handler) idpIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.SSO.All(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, p := range list {
		data = append(data, p.JSON(h.Cfg.AppURL()))
	}
	httpx.WriteData(w, http.StatusOK, data)
}

func (h Handler) idpShow(w http.ResponseWriter, r *http.Request) {
	p := h.idpParam(w, r)
	if p == nil {
		return
	}
	httpx.WriteJSON(w, http.StatusOK, p.JSON(h.Cfg.AppURL()))
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
	httpx.WriteJSON(w, http.StatusCreated, p.JSON(h.Cfg.AppURL()))
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
	httpx.WriteJSON(w, http.StatusOK, updated.JSON(h.Cfg.AppURL()))
}

func (h Handler) idpDestroy(w http.ResponseWriter, r *http.Request) {
	p := h.idpParam(w, r)
	if p == nil {
		return
	}
	if err := h.SSO.Delete(r.Context(), p.ID); err != nil {
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// testResult is the outcome of a connection test.
type testResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
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
		httpx.WriteJSON(w, status, testResult{Status: "error", Message: msg, Error: code})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, testResult{Status: "ok"})
}

func (h Handler) idpParam(w http.ResponseWriter, r *http.Request) *sso.IdentityProvider {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	p, err := h.SSO.ByID(r.Context(), id)
	if err != nil || p == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	return p
}

func decodeIdPWrite(w http.ResponseWriter, r *http.Request, create bool) (sso.IdPWrite, bool) {
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		httpx.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return sso.IdPWrite{}, false
	}
	var in sso.IdPWrite
	in.Name, _ = raw["name"].(string)
	in.Slug, _ = raw["slug"].(string)
	in.Preset, _ = raw["preset"].(string)
	in.DefaultRole, _ = raw["default_role"].(string)
	if v, ok := raw["enabled"]; ok {
		b := httpx.AsBool(v)
		in.Enabled = &b
	}
	if v, ok := raw["allow_jit"]; ok {
		b := httpx.AsBool(v)
		in.AllowJIT = &b
	}
	if v, ok := raw["sync_role_on_login"]; ok {
		b := httpx.AsBool(v)
		in.SyncRoleOnLogin = &b
	}
	if v, ok := raw["link_by_email"]; ok {
		b := httpx.AsBool(v)
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
			httpx.WriteValidation(w, errs)
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
			httpx.WriteValidation(w, map[string][]string{"slug": {e.Message}})
			return
		}
		if e.Status == 422 && e.Code == "slug_taken" {
			httpx.WriteValidation(w, map[string][]string{"slug": {e.Message}})
			return
		}
		httpx.WriteCoded(w, e.Status, e.Message, e.Code)
		return
	}
	httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
}
