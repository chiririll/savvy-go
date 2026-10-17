package space

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"savvy-go/internal/auth/password"
	"strconv"
	"strings"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/mail"
	"savvy-go/internal/mail/smtp"
	"savvy-go/internal/spaces"

	"github.com/go-chi/chi/v5"
)

func (h Handler) invitationsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Spaces.Invitations(r.Context(), httpx.Scope(r).ID)
	if err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewInvitation))
}

// inviteMethods says how invitations may be made right now: by email, which
// needs working mail and a public URL, and by a link to copy, which an instance
// that requires verified addresses does not allow. When email is off, reason
// says why.
func (h Handler) inviteMethods(ctx context.Context) (email, link bool, reason string) {
	switch {
	case !h.Mail.Mailer().Enabled():
		reason, _ = h.Mail.OffWhy()
	case h.Cfg.AppURL() == "":
		reason = mail.NoAppURLMessage
	default:
		email = true
	}
	return email, !h.Login.EmailRequired(ctx), reason
}

func (h Handler) invitationMethods(w http.ResponseWriter, r *http.Request) {
	email, link, reason := h.inviteMethods(r.Context())
	httpx.WriteData(w, http.StatusOK, dto.InvitationMethods{Email: email, Link: link, EmailReason: reason})
}

// invitationsStore creates an invitation. With an address it is emailed and
// its link is never returned, so the link proves the mailbox; without one the
// link is returned to copy. The server decides which of the two is allowed.
func (h Handler) invitationsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
		Note  string `json:"note"`
		Role  string `json:"role"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	if len([]rune(strings.TrimSpace(body.Note))) > spaces.MaxInvitationNote {
		httpx.WriteValidation(w, map[string][]string{"note": {fmt.Sprintf("The note may not be longer than %d characters.", spaces.MaxInvitationNote)}})
		return
	}
	if !spaces.ValidRole(body.Role) {
		httpx.WriteValidation(w, map[string][]string{"role": {"The role must be admin, editor or viewer."}})
		return
	}
	body.Email = strings.TrimSpace(body.Email)
	if body.Email != "" && !httpx.LooksLikeEmail(body.Email) {
		httpx.WriteValidation(w, map[string][]string{"email": {"The email field must be a valid email address."}})
		return
	}
	byEmail, byLink, reason := h.inviteMethods(r.Context())
	switch {
	case body.Email != "" && !byEmail:
		httpx.WriteValidation(w, map[string][]string{"email": {"Invitations cannot be emailed. " + reason}})
		return
	case body.Email == "" && !byLink:
		httpx.WriteValidation(w, map[string][]string{"email": {"This server requires invitations by email. Enter the address to invite."}})
		return
	}
	token, id, err := h.Spaces.Invite(r.Context(), httpx.Scope(r).ID, httpx.User(r), body.Email, body.Note, body.Role)
	if err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	if body.Email == "" {
		httpx.WriteData(w, http.StatusCreated, dto.InvitationCreated{ID: id, Token: token})
		return
	}
	if err := h.emailInvitation(r, body.Email, body.Role, token); err != nil {
		// Nobody received it, so it must not hold the address in a cooldown.
		if derr := h.Spaces.DiscardInvitation(r.Context(), id); derr != nil {
			slog.Warn("discard unsent invitation", "err", derr)
		}
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.InvitationCreated{ID: id, EmailSent: true})
}

func (h Handler) emailInvitation(r *http.Request, to, role, token string) error {
	space, err := h.Spaces.Get(r.Context(), httpx.Scope(r).ID)
	if err != nil || space == nil {
		return errors.New("could not read the space")
	}
	link := h.Cfg.AppURL() + "/invite/" + token
	if err := h.Mail.Send(r.Context(), smtp.Invitation(to, httpx.User(r).Name, space.Name, role, link)); err != nil {
		return errors.New("Could not send the email: " + err.Error())
	}
	return nil
}

func (h Handler) invitationsDestroy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Spaces.Revoke(r.Context(), httpx.Scope(r).ID, id); err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// invitationPreview is public: the link holder may not have an account yet.
func (h Handler) invitationPreview(w http.ResponseWriter, r *http.Request) {
	p, err := h.Spaces.Preview(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewInvitationPreview(*p))
}

func (h Handler) invitationAccept(w http.ResponseWriter, r *http.Request) {
	space, err := h.Spaces.Accept(r.Context(), chi.URLParam(r, "token"), httpx.User(r))
	if err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewSpace(*space))
}

// invitationRegister creates an account through an invitation and signs it
// in. Password sign-in must be on.
func (h Handler) invitationRegister(w http.ResponseWriter, r *http.Request) {
	if h.Login.PasswordLoginDisabled(r.Context()) {
		httpx.WriteMessage(w, http.StatusForbidden, spaces.ErrRegistrationClosed.Error())
		return
	}
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	errs := map[string][]string{}
	if strings.TrimSpace(body.Name) == "" {
		errs["name"] = []string{"The name field is required."}
	}
	if !httpx.LooksLikeEmail(body.Email) {
		errs["email"] = []string{"The email field must be a valid email address."}
	}
	if password.TooShort(body.Password) {
		errs["password"] = []string{password.TooShortMessage}
	}
	if len(errs) > 0 {
		httpx.WriteValidation(w, errs)
		return
	}
	if existing, _ := h.Users.ByEmail(r.Context(), body.Email); existing != nil {
		httpx.WriteValidation(w, map[string][]string{"email": {"Sign in with this email and open the invitation link again."}})
		return
	}
	token := chi.URLParam(r, "token")
	u, _, err := h.Spaces.Register(r.Context(), h.Users, token, strings.TrimSpace(body.Name), body.Email, body.Password)
	if err != nil {
		httpx.WriteSpaceError(w, err)
		return
	}
	h.askToConfirmAddress(r.Context(), u)
	h.Login.Issue(w, r, u, http.StatusCreated, false)
}

// askToConfirmAddress emails a new account a confirmation link when the
// instance requires verified addresses and the invitation did not already
// prove theirs (one that was emailed does).
func (h Handler) askToConfirmAddress(ctx context.Context, u *auth.User) {
	if u.EmailVerified || !h.Login.EmailRequired(ctx) {
		return
	}
	if problem := h.Verify.Send(ctx, u); problem != "" {
		slog.Warn("send verification email", "problem", problem)
	}
}
