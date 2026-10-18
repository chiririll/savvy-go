package dto

import (
	"time"

	"savvy-go/internal/spaces"
)

// Space is a space as the API lists it, with the caller's role in it.
type Space struct {
	ID   int64  `json:"id"`
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Role string `json:"role"`
}

func NewSpace(s spaces.Space) Space {
	return Space{ID: s.ID, UUID: s.UUID, Name: s.Name, Role: s.Role}
}

// SpaceDetail is the space the request works in, with its limits.
type SpaceDetail struct {
	Space
	Quota int64 `json:"quota"`
	Size  int64 `json:"size"`
}

type Member struct {
	UserID    int64      `json:"userId"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	CreatedAt *time.Time `json:"createdAt"`
}

func NewMember(m spaces.Member) Member {
	return Member{UserID: m.UserID, Name: m.Name, Email: m.Email, Role: m.Role, CreatedAt: utc(m.CreatedAt)}
}

type Invitation struct {
	ID         int64      `json:"id"`
	Email      *string    `json:"email"`
	Note       *string    `json:"note"`
	Role       string     `json:"role"`
	Inviter    string     `json:"inviter"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	AcceptedAt *time.Time `json:"acceptedAt"`
	CreatedAt  *time.Time `json:"createdAt"`
}

func NewInvitation(i spaces.Invitation) Invitation {
	return Invitation{
		ID: i.ID, Email: i.Email, Note: i.Note, Role: i.Role, Inviter: i.Inviter,
		ExpiresAt: utc(i.ExpiresAt), AcceptedAt: utc(i.AcceptedAt), CreatedAt: utc(i.CreatedAt),
	}
}

// InvitationPreview is what the holder of a link sees before accepting.
type InvitationPreview struct {
	SpaceName   string    `json:"spaceName"`
	Role        string    `json:"role"`
	Inviter     string    `json:"inviter"`
	Email       *string   `json:"email"`
	ExpiresAt   time.Time `json:"expiresAt"`
	CanRegister bool      `json:"canRegister"`
}

func NewInvitationPreview(p spaces.InvitationPreview) InvitationPreview {
	return InvitationPreview{
		SpaceName: p.SpaceName, Role: p.Role, Inviter: p.Inviter, Email: p.Email,
		ExpiresAt: p.ExpiresAt.UTC(), CanRegister: p.CanRegister,
	}
}

// InvitationMethods says how invitations may be made right now.
type InvitationMethods struct {
	Email       bool   `json:"email"`
	Link        bool   `json:"link"`
	EmailReason string `json:"emailReason,omitempty"`
}

// InvitationCreated names the new invitation. Token is the link secret, given
// only when no email carried it.
type InvitationCreated struct {
	ID        int64  `json:"id"`
	Token     string `json:"token,omitempty"`
	EmailSent bool   `json:"emailSent"`
}

type AuditEntry struct {
	ID           int64      `json:"id"`
	Actor        string     `json:"actor"`
	Action       string     `json:"action"`
	TargetUserID *int64     `json:"targetUserId"`
	Details      string     `json:"details"`
	CreatedAt    *time.Time `json:"createdAt"`
}

func NewAuditEntry(e spaces.AuditEntry) AuditEntry {
	return AuditEntry{
		ID: e.ID, Actor: e.Actor, Action: e.Action, TargetUserID: e.TargetUserID, Details: e.Details,
		CreatedAt: utc(e.CreatedAt),
	}
}

// SpaceOverview is a space as the server admin lists it: metadata only.
type SpaceOverview struct {
	ID          int64      `json:"id"`
	UUID        string     `json:"uuid"`
	Name        string     `json:"name"`
	Members     int64      `json:"members"`
	Admins      int64      `json:"admins"`
	Size        int64      `json:"size"`
	Quota       int64      `json:"quota"`
	Unavailable string     `json:"unavailable"`
	CreatedAt   *time.Time `json:"createdAt"`
}

func NewSpaceOverview(o spaces.Overview) SpaceOverview {
	return SpaceOverview{
		ID: o.ID, UUID: o.UUID, Name: o.Name, Members: o.Members, Admins: o.Admins, Size: o.Size, Quota: o.Quota,
		Unavailable: o.Unavailable, CreatedAt: utc(o.CreatedAt),
	}
}
