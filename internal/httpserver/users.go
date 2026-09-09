package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/chiririll/savvy-plus/internal/auth"
	"github.com/go-chi/chi/v5"
)

func (s *Server) usersIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.users.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, u := range list {
		data = append(data, u.ResourceJSON())
	}
	writeData(w, http.StatusOK, data)
}

func (s *Server) usersShow(w http.ResponseWriter, r *http.Request) {
	u := s.userParam(w, r)
	if u == nil {
		return
	}
	writeData(w, http.StatusOK, u.ResourceJSON())
}

func (s *Server) usersStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string  `json:"name"`
		Email    string  `json:"email"`
		Password *string `json:"password"`
		Role     string  `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	errs := map[string][]string{}
	if strings.TrimSpace(body.Name) == "" {
		errs["name"] = []string{"The name field is required."}
	}
	if !looksLikeEmail(body.Email) {
		errs["email"] = []string{"The email field must be a valid email address."}
	}
	if body.Password != nil && *body.Password != "" && len(*body.Password) < 8 {
		errs["password"] = []string{"The password must be at least 8 characters."}
	}
	if body.Role != "" && !validRole(body.Role) {
		errs["role"] = []string{"The selected role is invalid."}
	}
	if existing, _ := s.users.ByEmail(r.Context(), body.Email); existing != nil {
		errs["email"] = []string{"The email has already been taken."}
	}
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	u, err := s.users.Create(r.Context(), strings.TrimSpace(body.Name), body.Email, body.Password, body.Role)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	payload := u.ResourceJSON()
	if u.IsInactive() {
		tok, exp, err := s.tokens.Issue(r.Context(), u)
		if err != nil {
			writeMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		payload["token"] = tok
		payload["expiresAt"] = exp.UTC().Format("2006-01-02T15:04:05.000000Z")
	}
	writeData(w, http.StatusCreated, payload)
}

func (s *Server) usersUpdate(w http.ResponseWriter, r *http.Request) {
	u := s.userParam(w, r)
	if u == nil {
		return
	}
	var body struct {
		Name     *string `json:"name"`
		Email    *string `json:"email"`
		Password *string `json:"password"`
		Role     *string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	if body.Role != nil && *body.Role != auth.RoleAdmin && u.IsAdmin() {
		n, _ := s.users.AdminCount(r.Context())
		if n <= 1 {
			writeMessage(w, http.StatusUnprocessableEntity, "Cannot demote the last admin.")
			return
		}
	}
	if body.Password != nil && *body.Password != "" {
		_ = s.tokens.RevokeActive(r.Context(), u.ID)
	}
	updated, err := s.users.Update(r.Context(), u.ID, body.Name, body.Email, body.Role, body.Password)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, updated.ResourceJSON())
}

func (s *Server) usersDestroy(w http.ResponseWriter, r *http.Request) {
	u := s.userParam(w, r)
	if u == nil {
		return
	}
	me := userFrom(r)
	if u.ID == me.ID {
		writeMessage(w, http.StatusUnprocessableEntity, "Cannot delete yourself.")
		return
	}
	if u.IsAdmin() {
		n, _ := s.users.AdminCount(r.Context())
		if n <= 1 {
			writeMessage(w, http.StatusUnprocessableEntity, "Cannot delete the last admin.")
			return
		}
	}
	if err := s.users.Delete(r.Context(), u.ID); err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) usersIssueToken(w http.ResponseWriter, r *http.Request) {
	u := s.userParam(w, r)
	if u == nil {
		return
	}
	me := userFrom(r)
	if u.ID == me.ID {
		writeMessage(w, http.StatusUnprocessableEntity, "You cannot reset your own password this way.")
		return
	}
	if u.IsSSOOnly {
		writeMessage(w, http.StatusUnprocessableEntity, "This user signs in with SSO and has no password.")
		return
	}
	tok, exp, err := s.tokens.Issue(r.Context(), u)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	payload := u.ResourceJSON()
	payload["token"] = tok
	payload["expiresAt"] = exp.UTC().Format("2006-01-02T15:04:05.000000Z")
	writeData(w, http.StatusOK, payload)
}

func (s *Server) userParam(w http.ResponseWriter, r *http.Request) *auth.User {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	u, err := s.users.ByID(r.Context(), id)
	if err != nil || u == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	return u
}

func validRole(role string) bool {
	switch role {
	case auth.RoleAdmin, auth.RoleReadWrite, auth.RoleReadOnly:
		return true
	default:
		return false
	}
}
