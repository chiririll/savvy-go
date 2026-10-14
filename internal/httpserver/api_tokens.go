package httpserver

import (
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
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		AllSpaces   bool     `json:"all_spaces"`
		SpaceIDs    []int64  `json:"space_ids"`
		ExpiresAt   *string  `json:"expires_at"`
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
	// A token never gets more than it was asked for: no permissions means it
	// can list its spaces and nothing else, and it must name its spaces. It
	// writes only where its owner may too: the request is limited by the
	// token's permissions and the owner's role in the space.
	perms, err := auth.NormalizePermissions(body.Permissions)
	if err != nil {
		errs["permissions"] = []string{err.Error() + "."}
	}
	var spaceIDs []int64
	switch {
	case body.AllSpaces && len(body.SpaceIDs) > 0:
		errs["space_ids"] = []string{"Choose all spaces or a list of spaces, not both."}
	case !body.AllSpaces && len(body.SpaceIDs) == 0:
		errs["space_ids"] = []string{"Choose the spaces this token may reach."}
	case !body.AllSpaces:
		for _, id := range body.SpaceIDs {
			if role, err := s.spaces.Role(r.Context(), id, u.ID); err != nil || role == "" {
				errs["space_ids"] = []string{"You can only choose spaces you belong to."}
				break
			}
		}
		spaceIDs = body.SpaceIDs
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
	raw, tok, err := s.apiTokens.Issue(r.Context(), u, name, auth.Grant{Permissions: perms, AllSpaces: body.AllSpaces, SpaceIDs: spaceIDs}, expires)
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
		if err == auth.ErrNotFound {
			writeMessage(w, http.StatusNotFound, "Not found.")
			return
		}
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeMessage(w, http.StatusOK, "API token revoked.")
}
