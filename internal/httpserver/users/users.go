// Package users serves the administration of users.
package users

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"savvy-go/internal/backup"
	"savvy-go/internal/config"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/apidocs"
	"savvy-go/internal/httpserver/mailing"
	"savvy-go/internal/httpserver/signin"
	"savvy-go/internal/spaces"
	"strconv"
	"strings"

	"savvy-go/internal/auth"
	"savvy-go/internal/mailer"

	"github.com/go-chi/chi/v5"
)

// Handler holds what the user administration needs.
type Handler struct {
	Cfg     config.Config
	Users   auth.Users
	Tokens  auth.PasswordTokens
	Spaces  spaces.Service
	Mail    *mailing.Service
	Backups backup.Service
	Login   signin.Service
}

func (h Handler) usersIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Users.All(r.Context())
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	data := make([]any, 0, len(list))
	for _, u := range list {
		data = append(data, u.ResourceJSON())
	}
	api.WriteData(w, http.StatusOK, data)
}

func (h Handler) usersShow(w http.ResponseWriter, r *http.Request) {
	u := h.userParam(w, r)
	if u == nil {
		return
	}
	api.WriteData(w, http.StatusOK, u.ResourceJSON())
}

type userStoreBody struct {
	Name     string  `json:"name"`
	Email    string  `json:"email"`
	Password *string `json:"password"`
	Role     string  `json:"role"`
	// An admin vouching for the address, e.g. when mail is not set up.
	EmailVerified bool `json:"emailVerified"`
	// Email the set-password link instead of returning it. Needs a user without a password.
	Send bool `json:"send"`
}

func (b userStoreBody) hasPassword() bool { return b.Password != nil && *b.Password != "" }

// validateNewUser returns the field errors of a new user; a mail problem comes back
// separately because it is not a field error.
func (h Handler) validateNewUser(r *http.Request, body userStoreBody) (errs map[string][]string, mailProblem string) {
	errs = map[string][]string{}
	if strings.TrimSpace(body.Name) == "" {
		errs["name"] = []string{"The name field is required."}
	}
	if !api.LooksLikeEmail(body.Email) {
		errs["email"] = []string{"The email field must be a valid email address."}
	}
	if body.hasPassword() && len(*body.Password) < 8 {
		errs["password"] = []string{"The password must be at least 8 characters."}
	}
	if body.Role != "" && !validRole(body.Role) {
		errs["role"] = []string{"The selected role is invalid."}
	}
	if existing, _ := h.Users.ByEmail(r.Context(), body.Email); existing != nil {
		errs["email"] = []string{"The email has already been taken."}
	}
	switch {
	case body.Send && body.hasPassword():
		errs["send"] = []string{"A user with a password needs no link."}
	case body.Send:
		mailProblem = h.linkMailProblem(r)
	case !body.hasPassword() && !body.EmailVerified && h.Login.EmailRequired(r.Context()):
		// A link handed over by the admin proves nothing about the mailbox.
		errs["send"] = []string{"Email verification is required, so the link has to be emailed."}
	}
	return errs, mailProblem
}

// attachSetupLink issues the set-password token of a user without a password
// and either returns it in the payload or emails it.
func (h Handler) attachSetupLink(r *http.Request, u *auth.User, send bool, payload map[string]any) error {
	tok, exp, err := h.Tokens.Issue(r.Context(), u)
	if err != nil {
		return err
	}
	payload["expiresAt"] = exp.UTC().Format("2006-01-02T15:04:05.000000Z")
	switch {
	case !send:
		payload["token"] = tok
	case h.Mail.Send(r.Context(), mailer.PasswordReset(u.Email, u.Name, h.Cfg.URL()+"/set-password/"+tok, exp)) != nil:
		// The user exists but nobody saw the link: do not leave it valid. A reset can try again.
		_ = h.Tokens.RevokeActive(r.Context(), u.ID)
		payload["emailError"] = "Could not send the email."
	default:
		payload["emailSent"] = true
	}
	return nil
}

func (h Handler) usersStore(w http.ResponseWriter, r *http.Request) {
	var body userStoreBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	errs, mailProblem := h.validateNewUser(r, body)
	if mailProblem != "" {
		api.WriteMessage(w, http.StatusUnprocessableEntity, mailProblem)
		return
	}
	if len(errs) > 0 {
		api.WriteValidation(w, errs)
		return
	}
	u, err := h.Users.Create(r.Context(), strings.TrimSpace(body.Name), body.Email, body.Password, body.Role)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if body.EmailVerified {
		if err := h.Users.SetEmailVerified(r.Context(), u.ID, true); err != nil {
			api.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
		u.EmailVerified = true
	}
	payload := u.ResourceJSON()
	if u.IsInactive() {
		if err := h.attachSetupLink(r, u, body.Send, payload); err != nil {
			api.WriteMessage(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	api.WriteData(w, http.StatusCreated, payload)
}

func (h Handler) usersUpdate(w http.ResponseWriter, r *http.Request) {
	u := h.userParam(w, r)
	if u == nil {
		return
	}
	var body struct {
		Name     *string `json:"name"`
		Email    *string `json:"email"`
		Password *string `json:"password"`
		Role     *string `json:"role"`
		// An admin's word that the address is theirs; without it a new address is unverified.
		EmailVerified *bool `json:"emailVerified"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"email": {"The given data was invalid."}})
		return
	}
	if body.Role != nil && *body.Role != auth.RoleAdmin && u.IsAdmin() {
		n, _ := h.Users.AdminCount(r.Context())
		if n <= 1 {
			api.WriteMessage(w, http.StatusUnprocessableEntity, "Cannot demote the last admin.")
			return
		}
	}
	if body.Role != nil {
		if !validRole(*body.Role) {
			api.WriteValidation(w, map[string][]string{"role": {"The selected role is invalid."}})
			return
		}
		// P9: whoever administers a space stays able to.
		if err := h.Spaces.SetRoleAllowed(r.Context(), u, *body.Role); err != nil {
			api.WriteMessage(w, http.StatusUnprocessableEntity, "This user administers a space; make someone else its admin first.")
			return
		}
	}
	if body.Password != nil && *body.Password != "" {
		_ = h.Tokens.RevokeActive(r.Context(), u.ID)
	}
	if body.Role != nil && u.IsGuest() && *body.Role != auth.RoleGuest {
		_ = h.Spaces.Audit(r.Context(), api.User(r), "promote_guest", nil, &u.ID, *body.Role)
	}
	updated, err := h.Users.Update(r.Context(), u.ID, body.Name, body.Email, body.Role, body.Password, body.EmailVerified)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, updated.ResourceJSON())
}

func (h Handler) usersDestroy(w http.ResponseWriter, r *http.Request) {
	u := h.userParam(w, r)
	if u == nil {
		return
	}
	me := api.User(r)
	if u.ID == me.ID {
		api.WriteMessage(w, http.StatusUnprocessableEntity, "Cannot delete yourself.")
		return
	}
	if u.IsAdmin() {
		n, _ := h.Users.AdminCount(r.Context())
		if n <= 1 {
			api.WriteMessage(w, http.StatusUnprocessableEntity, "Cannot delete the last admin.")
			return
		}
	}
	if err := h.Spaces.DeleteUser(r.Context(), h.Users, u, h.Backups); err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) usersIssueToken(w http.ResponseWriter, r *http.Request) {
	u := h.userParam(w, r)
	if u == nil {
		return
	}
	me := api.User(r)
	if u.ID == me.ID {
		api.WriteMessage(w, http.StatusUnprocessableEntity, "You cannot reset your own password this way.")
		return
	}
	if u.IsSSOOnly {
		api.WriteMessage(w, http.StatusUnprocessableEntity, "This user signs in with SSO and has no password.")
		return
	}
	// The body is optional: {"send": true} emails the link instead of returning it.
	var body struct {
		Send bool `json:"send"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		api.WriteValidation(w, map[string][]string{"body": {"The given data was invalid."}})
		return
	}
	if body.Send {
		if problem := h.linkMailProblem(r); problem != "" {
			api.WriteMessage(w, http.StatusUnprocessableEntity, problem)
			return
		}
	}
	tok, exp, err := h.Tokens.Issue(r.Context(), u)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	payload := u.ResourceJSON()
	payload["expiresAt"] = exp.UTC().Format("2006-01-02T15:04:05.000000Z")
	if !body.Send {
		payload["token"] = tok
		api.WriteData(w, http.StatusOK, payload)
		return
	}
	if err := h.Mail.Send(r.Context(), mailer.PasswordReset(u.Email, u.Name, h.Cfg.URL()+"/set-password/"+tok, exp)); err != nil {
		// The link was never seen by anyone: do not leave it valid.
		_ = h.Tokens.RevokeActive(r.Context(), u.ID)
		api.WriteMessage(w, http.StatusUnprocessableEntity, "Could not send the email: "+err.Error())
		return
	}
	payload["emailSent"] = true
	api.WriteData(w, http.StatusOK, payload)
}

// linkMailProblem says why a set-password link cannot be emailed right now; ""
// when it can.
func (h Handler) linkMailProblem(r *http.Request) string {
	switch {
	case !h.Mail.Mailer().Enabled():
		return h.Mail.OffReason()
	case h.Login.PasswordLoginDisabled(r.Context()):
		return "Password sign-in is switched off, so a set-password link is of no use."
	case h.Cfg.URL() == "":
		return mailing.NoAppURLMessage
	}
	return ""
}

func (h Handler) userParam(w http.ResponseWriter, r *http.Request) *auth.User {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	u, err := h.Users.ByID(r.Context(), id)
	if err != nil || u == nil || u.Deleted {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
		return nil
	}
	return u
}

func validRole(role string) bool { return auth.ValidRole(role) }

//go:embed openapi.yaml
var spec []byte

func (h Handler) Routes(r chi.Router) {
	apidocs.Add(spec)
	r.Get("/users", h.usersIndex)
	r.Get("/users/{id}", h.usersShow)
	r.Post("/users", h.usersStore)
	r.Post("/users/{id}/password-token", h.usersIssueToken)
	r.Put("/users/{id}", h.usersUpdate)
	r.Patch("/users/{id}", h.usersUpdate)
	r.Delete("/users/{id}", h.usersDestroy)
}
