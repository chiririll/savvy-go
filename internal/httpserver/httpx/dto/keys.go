package dto

import (
	"time"

	"savvy-go/internal/transfers"
)

// SigningKey is the public half of a key that signs transfers between spaces.
type SigningKey struct {
	KID       string `json:"kid"`
	PublicKey string `json:"publicKey"`
}

type TrustedKey struct {
	SigningKey
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"createdAt"`
}

func NewTrustedKey(k transfers.TrustedKey) TrustedKey {
	return TrustedKey{SigningKey: SigningKey{KID: k.KID, PublicKey: k.PublicKey}, Name: k.Name, CreatedAt: utc(k.CreatedAt)}
}

// Keys are this instance's own key and the keys of instances it trusts.
type Keys struct {
	Own     SigningKey   `json:"own"`
	Trusted []TrustedKey `json:"trusted"`
}
