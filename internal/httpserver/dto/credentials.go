package dto

import (
	"time"

	"savvy-go/internal/auth"
)

type WebAuthnCred struct {
	ID         int64   `json:"id"`
	Name       *string `json:"name"`
	AAGUID     *string `json:"aaguid"`
	LastUsedAt *string `json:"lastUsedAt"`
	CreatedAt  *string `json:"createdAt"`
}

func NewWebAuthnCred(c auth.WebAuthnCred) WebAuthnCred {
	return WebAuthnCred{ID: c.ID, Name: c.Name, AAGUID: c.AAGUID, LastUsedAt: c.LastUsedAt, CreatedAt: c.CreatedAt}
}

func APIToken(t auth.APIToken) map[string]any {
	return map[string]any{
		"id": t.ID, "name": t.Name, "prefix": t.Prefix, "kind": t.Kind,
		"permissions": t.Permissions, "all_spaces": t.AllSpaces, "space_ids": t.SpaceIDs,
		"expires_at": timePtr(t.ExpiresAt), "last_used_at": timePtr(t.LastUsedAt), "created_at": timePtr(t.CreatedAt),
	}
}

func timePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
