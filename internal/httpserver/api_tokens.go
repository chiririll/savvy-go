package httpserver

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/dto"

	"github.com/go-chi/chi/v5"
)

func (s *Server) apiTokensIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.apiTokens.List(r.Context(), userFrom(r).ID)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, t := range list {
		data = append(data, dto.APIToken(t))
	}
	writeData(w, http.StatusOK, data)
}

func (s *Server) apiTokensStore(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var body struct {
		Name      string  `json:"name"`
		Scope     string  `json:"scope"`
		ExpiresAt *string `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	errs := map[string][]string{}
	name := strings.TrimSpace(body.Name)
	if name == "" || len(name) > 100 {
		errs["name"] = []string{"The name is required and must be at most 100 characters."}
	}
	scope := body.Scope
	if scope == "" {
		scope = auth.APIScopeRead
	}
	// A read-write token still writes only where its owner may: the request
	// is limited by the token scope and the owner's role in the space.
	if !auth.ValidAPIScope(scope) {
		errs["scope"] = []string{"The scope must be read or read-write."}
	}
	var expires *time.Time
	if body.ExpiresAt != nil && *body.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *body.ExpiresAt)
		if err != nil || !t.After(time.Now()) {
			errs["expires_at"] = []string{"The expiration must be a future date (RFC 3339)."}
		} else {
			t = t.UTC()
			expires = &t
		}
	}
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	raw, tok, err := s.apiTokens.Issue(r.Context(), u, name, scope, expires)
	if err == auth.ErrTokenLimit {
		writeMessage(w, http.StatusUnprocessableEntity, "You can create at most 50 API tokens.")
		return
	}
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := dto.APIToken(*tok)
	out["token"] = raw
	writeDataMsg(w, http.StatusCreated, out, "API token created.")
}

func (s *Server) apiTokensDestroy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := s.apiTokens.Revoke(r.Context(), userFrom(r).ID, id); err != nil {
		if err == sql.ErrNoRows {
			writeMessage(w, http.StatusNotFound, "Not found.")
			return
		}
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeMessage(w, http.StatusOK, "API token revoked.")
}
