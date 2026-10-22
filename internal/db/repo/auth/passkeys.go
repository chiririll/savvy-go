package auth

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
)

// Passkey is a registered credential as the settings page lists it.
type Passkey struct {
	ID                    int64
	Name, AAGUID          *string
	LastUsedAt, CreatedAt *time.Time
}

func (r Repo) Passkeys(ctx context.Context, userID int64) ([]Passkey, error) {
	rows, err := r.q().ListWebAuthnCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Passkey, len(rows))
	for i, row := range rows {
		out[i] = Passkey{ID: row.ID, Name: optString(row.Name), AAGUID: optString(row.Aaguid),
			LastUsedAt: optTime(row.LastUsedAt), CreatedAt: optTime(row.CreatedAt)}
	}
	return out, nil
}

func (r Repo) CountPasskeys(ctx context.Context, userID int64) (int, error) {
	n, err := r.q().CountWebAuthnCredentials(ctx, userID)
	return int(n), err
}

// PasskeyRecords are the serialized credentials the WebAuthn library needs.
func (r Repo) PasskeyRecords(ctx context.Context, userID int64) ([]string, error) {
	return r.q().ListWebAuthnRecords(ctx, userID)
}

type NewPasskey struct {
	UserID                             int64
	CredentialID, Name, AAGUID, Record string
	Counter                            int64
	Now                                time.Time
}

func (r Repo) InsertPasskey(ctx context.Context, p NewPasskey) (int64, error) {
	res, err := r.q().InsertWebAuthnCredential(ctx, sqlc.InsertWebAuthnCredentialParams{
		UserID: p.UserID, CredentialID: p.CredentialID, Name: db.NS(p.Name), Aaguid: db.NS(p.AAGUID),
		Record: p.Record, Counter: p.Counter, CreatedAt: db.TimeNS(p.Now), UpdatedAt: db.TimeNS(p.Now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// RenamePasskey and DeletePasskey report whether the user had such a passkey.
func (r Repo) RenamePasskey(ctx context.Context, userID, id int64, name string, now time.Time) (bool, error) {
	res, err := r.q().RenameWebAuthnCredential(ctx, sqlc.RenameWebAuthnCredentialParams{
		Name: db.NS(name), UpdatedAt: db.TimeNS(now), ID: id, UserID: userID,
	})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (r Repo) DeletePasskey(ctx context.Context, userID, id int64) (bool, error) {
	res, err := r.q().DeleteWebAuthnCredential(ctx, sqlc.DeleteWebAuthnCredentialParams{ID: id, UserID: userID})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// UpdatePasskeyAfterLogin stores the credential record and sign count the
// authenticator reported, and marks the passkey as used.
func (r Repo) UpdatePasskeyAfterLogin(
	ctx context.Context,
	credentialID, record string,
	counter int64,
	now time.Time,
) error {
	return r.q().UpdateWebAuthnCredentialAfterLogin(ctx, sqlc.UpdateWebAuthnCredentialAfterLoginParams{
		Record: record, Counter: counter, LastUsedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), CredentialID: credentialID,
	})
}

// PasskeyUserID is the owner of a credential, or nil if it is unknown.
func (r Repo) PasskeyUserID(ctx context.Context, credentialID string) (*int64, error) {
	return found(r.q().GetWebAuthnUserIDByCredential(ctx, credentialID))
}

// PasskeyChallenge is a registration or login ceremony in progress.
type PasskeyChallenge struct {
	UserID  int64 // 0 for a login that does not name the user
	Options string
}

func (r Repo) InsertPasskeyChallenge(
	ctx context.Context,
	userID *int64,
	hash, typ, options string,
	expires, now time.Time,
) error {
	return r.q().InsertWebAuthnChallenge(ctx, sqlc.InsertWebAuthnChallengeParams{
		UserID: db.NullInt64(userID), TokenHash: hash, Type: typ, Options: options,
		ExpiresAt: db.FormatTime(expires), CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
}

// TakePasskeyChallenge returns an open challenge of the type and marks it
// used in one statement, so of two racing calls only one gets it. A challenge
// that is unknown, used or expired is nil; a database error is returned.
func (r Repo) TakePasskeyChallenge(ctx context.Context, hash, typ string, now time.Time) (*PasskeyChallenge, error) {
	row, err := r.q().TakeWebAuthnChallenge(ctx, sqlc.TakeWebAuthnChallengeParams{
		ConsumedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
		TokenHash: hash, Type: typ, ExpiresAt: db.FormatTime(now),
	})
	if err != nil {
		if err == db.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &PasskeyChallenge{UserID: row.UserID.Int64, Options: row.Options}, nil
}
