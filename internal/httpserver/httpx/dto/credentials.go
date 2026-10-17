package dto

import (
	"time"

	"savvy-go/internal/auth"
)

type WebAuthnCred struct {
	ID         int64      `json:"id"`
	Name       *string    `json:"name"`
	AAGUID     *string    `json:"aaguid"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	CreatedAt  *time.Time `json:"createdAt"`
}

func NewWebAuthnCred(c auth.WebAuthnCred) WebAuthnCred {
	return WebAuthnCred{
		ID:         c.ID,
		Name:       c.Name,
		AAGUID:     c.AAGUID,
		LastUsedAt: utc(c.LastUsedAt),
		CreatedAt:  utc(c.CreatedAt),
	}
}

// APIToken keeps snake_case keys, unlike the rest of the package: clients send
// the same names (all_spaces, space_ids, expires_at) when they create or edit a
// token, and read them back unchanged.
type APIToken struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Prefix      string     `json:"prefix"`
	Kind        string     `json:"kind"`
	Permissions []string   `json:"permissions"`
	AllSpaces   bool       `json:"all_spaces"`
	SpaceIDs    []int64    `json:"space_ids"`
	ExpiresAt   *time.Time `json:"expires_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   *time.Time `json:"created_at"`
}

func NewAPIToken(t auth.APIToken) APIToken {
	perms, ids := t.Permissions, t.SpaceIDs
	if perms == nil {
		perms = []string{}
	}
	if ids == nil {
		ids = []int64{}
	}
	return APIToken{
		ID: t.ID, Name: t.Name, Prefix: t.Prefix, Kind: t.Kind, Permissions: perms, AllSpaces: t.AllSpaces, SpaceIDs: ids,
		ExpiresAt: utc(t.ExpiresAt), LastUsedAt: utc(t.LastUsedAt), CreatedAt: utc(t.CreatedAt),
	}
}

// CreatedAPIToken is a new token with its secret, shown only once.
type CreatedAPIToken struct {
	APIToken
	Token string `json:"token"`
}

func NewCreatedAPIToken(t auth.APIToken, secret string) CreatedAPIToken {
	return CreatedAPIToken{APIToken: NewAPIToken(t), Token: secret}
}

// TokenKindCapabilities is what one kind of API token can be given: the parts
// of a space, and what each preset grants over them.
type TokenKindCapabilities struct {
	Resources []string            `json:"resources"`
	Presets   map[string][]string `json:"presets"`
}

// TokenCapabilities is what the screens that configure a token offer.
type TokenCapabilities struct {
	Levels  []string                         `json:"levels"`
	Presets []string                         `json:"presets"`
	Kinds   map[string]TokenKindCapabilities `json:"kinds"`
}
