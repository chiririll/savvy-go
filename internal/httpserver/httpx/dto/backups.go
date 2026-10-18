package dto

import (
	"time"

	"savvy-go/internal/backup"
	"savvy-go/internal/signing"
)

// Backup status: "current" is a backup signed by this server (or a key it
// trusts), "unsigned" one made elsewhere or edited, and "invalid" one that cannot
// be read.
type Backup struct {
	Filename   string     `json:"filename"`
	Size       int64      `json:"size"`
	Kind       string     `json:"kind"`
	Note       *string    `json:"note"`
	AppVersion *string    `json:"appVersion"`
	SpaceName  string     `json:"spaceName,omitempty"`
	Status     string     `json:"status"`
	Signature  string     `json:"signature"`
	Restorable bool       `json:"restorable"`
	CreatedAt  *time.Time `json:"createdAt"`
}

func NewBackup(b backup.Entry) Backup {
	status := "invalid"
	switch {
	case !b.Valid:
	case b.Signature == signing.SignedHere || b.Signature == signing.SignedTrusted:
		status = "current"
	default:
		status = "unsigned"
	}
	return Backup{
		Filename: b.Filename, Size: b.Size, Kind: b.Kind, Note: b.Note, AppVersion: b.AppVersion,
		SpaceName: b.SpaceName, Status: status, Signature: string(b.Signature), Restorable: b.Valid,
		CreatedAt: utc(b.CreatedAt),
	}
}
