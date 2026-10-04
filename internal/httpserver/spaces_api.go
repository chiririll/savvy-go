package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver/dto"
)

func spaceJSON(space domain.Space) map[string]any {
	return map[string]any{"id": space.ID, "uuid": space.UUID, "name": space.Name, "role": space.Role}
}

// writeSpaceError maps space rules to 422 and forbidden ones to 403.
func writeSpaceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotMember):
		writeMessage(w, http.StatusNotFound, "Not a member of this space.")
	case errors.Is(err, domain.ErrInvitationInvalid):
		writeMessage(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrRegistrationClosed):
		writeMessage(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrGuestCannotAdmin), errors.Is(err, domain.ErrLastAdmin), errors.Is(err, domain.ErrLastMember),
		errors.Is(err, domain.ErrSpaceLimit), errors.Is(err, domain.ErrInvitationEmail):
		writeMessage(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeMessage(w, http.StatusInternalServerError, err.Error())
	}
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeValidation(w, map[string][]string{"body": {"The given data was invalid."}})
		return false
	}
	return true
}

func spaceName(w http.ResponseWriter, raw string) (string, bool) {
	name := strings.TrimSpace(raw)
	if name == "" || len(name) > 100 {
		writeValidation(w, map[string][]string{"name": {"The name is required and must be at most 100 characters."}})
		return "", false
	}
	return name, true
}

// --- spaces ----------------------------------------------------------------

func (s *Server) spacesStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	name, ok := spaceName(w, body.Name)
	if !ok {
		return
	}
	space, err := s.spaces.CreateOwned(r.Context(), name, userFrom(r))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeData(w, http.StatusCreated, spaceJSON(*space))
}

func (s *Server) spaceShow(w http.ResponseWriter, r *http.Request) {
	space := s.currentSpace(w, r)
	if space == nil {
		return
	}
	space.Role = sp(r).role
	out := spaceJSON(*space)
	out["quota"] = s.spaces.Quota(r.Context(), space.ID)
	out["size"], _ = s.store.SpaceSize(r.Context(), space.ID)
	writeData(w, http.StatusOK, out)
}

func (s *Server) spaceUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	name, ok := spaceName(w, body.Name)
	if !ok {
		return
	}
	if err := s.spaces.Rename(r.Context(), sp(r).id, name); err != nil {
		writeSpaceError(w, err)
		return
	}
	s.spaceShow(w, r)
}

func (s *Server) spaceDestroy(w http.ResponseWriter, r *http.Request) {
	space := s.currentSpace(w, r)
	if space == nil {
		return
	}
	if err := s.spaces.Delete(r.Context(), *space, s.backups); err != nil {
		writeSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- members ---------------------------------------------------------------

func (s *Server) membersIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.spaces.Members(r.Context(), sp(r).id)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, func(m domain.Member) map[string]any {
		return map[string]any{"userId": m.UserID, "name": m.Name, "email": m.Email, "role": m.Role, "createdAt": timeOrNil(m.CreatedAt)}
	}))
}

// memberParam is the user named by {user}, if they belong to the space.
func (s *Server) memberParam(w http.ResponseWriter, r *http.Request) *auth.User {
	id, err := strconv.ParseInt(chi.URLParam(r, "user"), 10, 64)
	if err == nil {
		if role, _ := s.spaces.Role(r.Context(), sp(r).id, id); role != "" {
			if u, _ := s.users.ByID(r.Context(), id); u != nil {
				return u
			}
		}
	}
	writeMessage(w, http.StatusNotFound, "Not a member of this space.")
	return nil
}

func (s *Server) membersUpdate(w http.ResponseWriter, r *http.Request) {
	u := s.memberParam(w, r)
	if u == nil {
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if !domain.ValidSpaceRole(body.Role) {
		writeValidation(w, map[string][]string{"role": {"The role must be admin, editor or viewer."}})
		return
	}
	if err := s.spaces.ChangeRole(r.Context(), sp(r).id, u, body.Role); err != nil {
		writeSpaceError(w, err)
		return
	}
	s.membersIndex(w, r)
}

func (s *Server) membersDestroy(w http.ResponseWriter, r *http.Request) {
	u := s.memberParam(w, r)
	if u == nil {
		return
	}
	if err := s.spaces.RemoveMember(r.Context(), sp(r).id, u.ID); err != nil {
		writeSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) spaceLeave(w http.ResponseWriter, r *http.Request) {
	if err := s.spaces.RemoveMember(r.Context(), sp(r).id, userFrom(r).ID); err != nil {
		writeSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- invitations -----------------------------------------------------------

func (s *Server) invitationsIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.spaces.Invitations(r.Context(), sp(r).id)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, func(i domain.Invitation) map[string]any {
		return map[string]any{
			"id": i.ID, "email": i.Email, "role": i.Role, "inviter": i.Inviter,
			"expiresAt": timeOrNil(i.ExpiresAt), "acceptedAt": timeOrNil(i.AcceptedAt), "createdAt": timeOrNil(i.CreatedAt),
		}
	}))
}

func (s *Server) invitationsStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if !domain.ValidSpaceRole(body.Role) {
		writeValidation(w, map[string][]string{"role": {"The role must be admin, editor or viewer."}})
		return
	}
	if body.Email != "" && !looksLikeEmail(body.Email) {
		writeValidation(w, map[string][]string{"email": {"The email field must be a valid email address."}})
		return
	}
	token, err := s.spaces.Invite(r.Context(), sp(r).id, userFrom(r), body.Email, body.Role)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeData(w, http.StatusCreated, map[string]any{"token": token})
}

func (s *Server) invitationsDestroy(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := s.spaces.Revoke(r.Context(), sp(r).id, id); err != nil {
		writeSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// invitationPreview is public: the link holder may not have an account yet.
func (s *Server) invitationPreview(w http.ResponseWriter, r *http.Request) {
	p, err := s.spaces.Preview(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"spaceName": p.SpaceName, "role": p.Role, "inviter": p.Inviter, "email": p.Email,
		"expiresAt": p.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z"), "canRegister": p.CanRegister,
	})
}

func (s *Server) invitationAccept(w http.ResponseWriter, r *http.Request) {
	space, err := s.spaces.Accept(r.Context(), chi.URLParam(r, "token"), userFrom(r))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeData(w, http.StatusOK, spaceJSON(*space))
}

// invitationRegister creates an account through an invitation and signs it
// in. Password sign-in must be on.
func (s *Server) invitationRegister(w http.ResponseWriter, r *http.Request) {
	if s.passwordLoginDisabled(r) {
		writeMessage(w, http.StatusForbidden, domain.ErrRegistrationClosed.Error())
		return
	}
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	errs := map[string][]string{}
	if strings.TrimSpace(body.Name) == "" {
		errs["name"] = []string{"The name field is required."}
	}
	if !looksLikeEmail(body.Email) {
		errs["email"] = []string{"The email field must be a valid email address."}
	}
	if len(body.Password) < 8 {
		errs["password"] = []string{"The password must be at least 8 characters."}
	}
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}
	if existing, _ := s.users.ByEmail(r.Context(), body.Email); existing != nil {
		writeValidation(w, map[string][]string{"email": {"Sign in with this email and open the invitation link again."}})
		return
	}
	u, _, err := s.spaces.Register(r.Context(), s.users, chi.URLParam(r, "token"), strings.TrimSpace(body.Name), body.Email, body.Password)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	s.issueSession(w, r, u, http.StatusCreated, false)
}

// --- space settings and audit ----------------------------------------------

func (s *Server) spaceSettingsIndex(w http.ResponseWriter, r *http.Request) {
	all, err := sp(r).settings.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, all)
}

func (s *Server) spaceAudit(w http.ResponseWriter, r *http.Request) {
	list, err := s.spaces.AuditLog(r.Context(), sp(r).id)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, auditJSON))
}

func auditJSON(e domain.AuditEntry) map[string]any {
	return map[string]any{"id": e.ID, "actor": e.Actor, "action": e.Action, "targetUserId": e.TargetUserID,
		"details": e.Details, "createdAt": timeOrNil(e.CreatedAt)}
}

// --- server admin ----------------------------------------------------------

// adminSpacesIndex is the overview of every space: metadata only (P5).
func (s *Server) adminSpacesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.spaces.Overview(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, func(o domain.Overview) map[string]any {
		return map[string]any{"id": o.ID, "uuid": o.UUID, "name": o.Name, "members": o.Members, "admins": o.Admins,
			"size": o.Size, "quota": o.Quota, "unavailable": o.Unavailable, "createdAt": timeOrNil(o.CreatedAt)}
	}))
}

func (s *Server) adminSpaceParam(w http.ResponseWriter, r *http.Request) *domain.Space {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err == nil {
		if space, _ := s.spaces.Get(r.Context(), id); space != nil {
			return space
		}
	}
	writeMessage(w, http.StatusNotFound, "Space not found.")
	return nil
}

func (s *Server) adminSpaceUpdate(w http.ResponseWriter, r *http.Request) {
	space := s.adminSpaceParam(w, r)
	if space == nil {
		return
	}
	var body struct {
		QuotaBytes *int64 `json:"quota_bytes"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	var bytes int64
	if body.QuotaBytes != nil {
		bytes = *body.QuotaBytes
	}
	if err := s.spaces.SetQuota(r.Context(), userFrom(r), space.ID, bytes); err != nil {
		writeSpaceError(w, err)
		return
	}
	s.adminSpacesIndex(w, r)
}

func (s *Server) adminSpaceDestroy(w http.ResponseWriter, r *http.Request) {
	space := s.adminSpaceParam(w, r)
	if space == nil {
		return
	}
	if err := s.spaces.Delete(r.Context(), *space, s.backups); err != nil {
		writeSpaceError(w, err)
		return
	}
	_ = s.spaces.Audit(r.Context(), userFrom(r), "delete_space", &space.ID, nil, space.Name)
	w.WriteHeader(http.StatusNoContent)
}

// adminSpaceAssignAdmin makes a user (the caller included) admin of a space,
// outside its admins' control; the space's admins see it in their audit log.
func (s *Server) adminSpaceAssignAdmin(w http.ResponseWriter, r *http.Request) {
	space := s.adminSpaceParam(w, r)
	if space == nil {
		return
	}
	var body struct {
		UserID int64 `json:"user_id"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	u, _ := s.users.ByID(r.Context(), body.UserID)
	if u == nil || u.Deleted {
		writeMessage(w, http.StatusNotFound, "User not found.")
		return
	}
	if err := s.spaces.AssignAdmin(r.Context(), userFrom(r), space.ID, u); err != nil {
		writeSpaceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) adminDeletedIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.backups.DeletedBackups()
	backupList(w, list, err)
}

// adminDeletedRestore brings a deleted space back as a new space owned by the
// calling server admin.
func (s *Server) adminDeletedRestore(w http.ResponseWriter, r *http.Request) {
	path, err := s.backups.DeletedPath(chi.URLParam(r, "name"))
	if err != nil {
		writeBackupError(w, err)
		return
	}
	space, err := s.backups.ImportSpace(r.Context(), s.spaces, path, "", userFrom(r), 0)
	if err != nil {
		writeBackupError(w, err)
		return
	}
	_ = s.spaces.Audit(r.Context(), userFrom(r), "restore_deleted_space", &space.ID, nil, space.Name)
	writeData(w, http.StatusCreated, spaceJSON(*space))
}

// timeOrNil renders a timestamp as RFC 3339 UTC, or null.
func timeOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
