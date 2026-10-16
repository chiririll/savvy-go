package spaceapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/mailer"
	"savvy-go/internal/mailing"
	"savvy-go/internal/spaces"

	"github.com/go-chi/chi/v5"
)

func (h Handler) invitationsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Spaces.Invitations(r.Context(), api.Scope(r).ID)
	if err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	api.WriteData(w, http.StatusOK, dto.Map(list, func(i spaces.Invitation) map[string]any {
		return map[string]any{
			"id": i.ID, "email": i.Email, "note": i.Note, "role": i.Role, "inviter": i.Inviter,
			"expiresAt": api.TimeOrNil(i.ExpiresAt), "acceptedAt": api.TimeOrNil(i.AcceptedAt), "createdAt": api.TimeOrNil(i.CreatedAt),
		}
	}))
}

// inviteMethods says how invitations may be made right now: by email, which
// needs working mail and a public URL, and by a link to copy, which an instance
// that requires verified addresses does not allow. When email is off, reason
// says why.
func (h Handler) inviteMethods(ctx context.Context) (email, link bool, reason string) {
	switch {
	case !h.Mail.Mailer().Enabled():
		reason, _ = h.Mail.OffWhy()
	case h.Cfg.URL() == "":
		reason = mailing.NoAppURLMessage
	default:
		email = true
	}
	return email, !h.Login.EmailRequired(ctx), reason
}

func (h Handler) invitationMethods(w http.ResponseWriter, r *http.Request) {
	email, link, reason := h.inviteMethods(r.Context())
	out := map[string]any{"email": email, "link": link}
	if reason != "" {
		out["emailReason"] = reason
	}
	api.WriteData(w, http.StatusOK, out)
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
	if !api.DecodeBody(w, r, &body) {
		return
	}
	if len([]rune(strings.TrimSpace(body.Note))) > spaces.MaxInvitationNote {
		api.WriteValidation(w, map[string][]string{"note": {fmt.Sprintf("The note may not be longer than %d characters.", spaces.MaxInvitationNote)}})
		return
	}
	if !spaces.ValidRole(body.Role) {
		api.WriteValidation(w, map[string][]string{"role": {"The role must be admin, editor or viewer."}})
		return
	}
	body.Email = strings.TrimSpace(body.Email)
	if body.Email != "" && !api.LooksLikeEmail(body.Email) {
		api.WriteValidation(w, map[string][]string{"email": {"The email field must be a valid email address."}})
		return
	}
	byEmail, byLink, reason := h.inviteMethods(r.Context())
	switch {
	case body.Email != "" && !byEmail:
		api.WriteValidation(w, map[string][]string{"email": {"Invitations cannot be emailed. " + reason}})
		return
	case body.Email == "" && !byLink:
		api.WriteValidation(w, map[string][]string{"email": {"This server requires invitations by email. Enter the address to invite."}})
		return
	}
	token, id, err := h.Spaces.Invite(r.Context(), api.Scope(r).ID, api.User(r), body.Email, body.Note, body.Role)
	if err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	if body.Email == "" {
		api.WriteData(w, http.StatusCreated, map[string]any{"id": id, "token": token, "emailSent": false})
		return
	}
	if err := h.emailInvitation(r, body.Email, body.Role, token); err != nil {
		// Nobody received it, so it must not hold the address in a cooldown.
		if derr := h.Spaces.DiscardInvitation(r.Context(), id); derr != nil {
			slog.Warn("discard unsent invitation", "err", derr)
		}
		api.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	api.WriteData(w, http.StatusCreated, map[string]any{"id": id, "emailSent": true})
}

func (h Handler) emailInvitation(r *http.Request, to, role, token string) error {
	space, err := h.Spaces.Get(r.Context(), api.Scope(r).ID)
	if err != nil || space == nil {
		return errors.New("could not read the space")
	}
	link := h.Cfg.URL() + "/invite/" + token
	if err := h.Mail.Send(r.Context(), mailer.Invitation(to, api.User(r).Name, space.Name, role, link)); err != nil {
		return errors.New("Could not send the email: " + err.Error())
	}
	return nil
}

// invitationsClear ends the cooldown of revoked and expired invitations, for
// one address or all of them. It is for server admins only.
func (h Handler) invitationsClear(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
	}
	if !api.DecodeBody(w, r, &body) {
		return
	}
	n, err := h.Spaces.ClearInvitationCooldowns(r.Context(), api.User(r), body.Email)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{"cleared": n})
}

func (h Handler) invitationsDestroy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Spaces.Revoke(r.Context(), api.Scope(r).ID, id); err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// invitationPreview is public: the link holder may not have an account yet.
func (h Handler) invitationPreview(w http.ResponseWriter, r *http.Request) {
	p, err := h.Spaces.Preview(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	api.WriteData(w, http.StatusOK, map[string]any{
		"spaceName": p.SpaceName, "role": p.Role, "inviter": p.Inviter, "email": p.Email,
		"expiresAt": p.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"), "canRegister": p.CanRegister,
	})
}

func (h Handler) invitationAccept(w http.ResponseWriter, r *http.Request) {
	space, err := h.Spaces.Accept(r.Context(), chi.URLParam(r, "token"), api.User(r))
	if err != nil {
		api.WriteSpaceError(w, err)
		return
	}
	api.WriteData(w, http.StatusOK, api.SpaceJSON(*space))
}

// invitationRegister creates an account through an invitation and signs it
// in. Password sign-in must be on.
func (h Handler) invitationRegister(w http.ResponseWriter, r *http.Request) {
	if h.Login.PasswordLoginDisabled(r.Context()) {
		api.WriteMessage(w, http.StatusForbidden, spaces.ErrRegistrationClosed.Error())
		return
	}
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !api.DecodeBody(w, r, &body) {
		return
	}
	errs := map[string][]string{}
	if strings.TrimSpace(body.Name) == "" {
		errs["name"] = []string{"The name field is required."}
	}
	if !api.LooksLikeEmail(body.Email) {
		errs["email"] = []string{"The email field must be a valid email address."}
	}
	if len(body.Password) < 8 {
		errs["password"] = []string{"The password must be at least 8 characters."}
	}
	if len(errs) > 0 {
		api.WriteValidation(w, errs)
		return
	}
	if existing, _ := h.Users.ByEmail(r.Context(), body.Email); existing != nil {
		api.WriteValidation(w, map[string][]string{"email": {"Sign in with this email and open the invitation link again."}})
		return
	}
	token := chi.URLParam(r, "token")
	u, _, err := h.Spaces.Register(r.Context(), h.Users, token, strings.TrimSpace(body.Name), body.Email, body.Password)
	if err != nil {
		api.WriteSpaceError(w, err)
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
