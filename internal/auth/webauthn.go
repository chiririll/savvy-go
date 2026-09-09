package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/chiririll/savvy-plus/internal/config"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	WebAuthnRegister = "registration"
	WebAuthnLogin    = "authentication"
	maxPasskeys      = 20
)

type WebAuthn struct {
	DB  *sql.DB
	Cfg config.Config
}

type waUser struct {
	user        *User
	credentials []webauthn.Credential
}

func (u waUser) WebAuthnID() []byte          { return []byte(fmt.Sprintf("%d", u.user.ID)) }
func (u waUser) WebAuthnName() string        { return u.user.Email }
func (u waUser) WebAuthnDisplayName() string { return u.user.Name }
func (u waUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

type WebAuthnCred struct {
	ID         int64
	Name       *string
	AAGUID     *string
	LastUsedAt *string
	CreatedAt  *string
}

func (c WebAuthnCred) JSON() map[string]any {
	return map[string]any{
		"id": c.ID, "name": c.Name, "aaguid": c.AAGUID,
		"last_used_at": c.LastUsedAt, "created_at": c.CreatedAt,
	}
}

func (w WebAuthn) engine() (*webauthn.WebAuthn, error) {
	u, err := url.Parse(w.Cfg.AppURL)
	if err != nil || u.Host == "" {
		u, _ = url.Parse("http://localhost:8080")
	}
	rpID := u.Hostname()
	if rpID == "" {
		rpID = "localhost"
	}
	origin := strings.TrimRight(w.Cfg.AppURL, "/")
	return webauthn.New(&webauthn.Config{
		RPDisplayName: "Savvy",
		RPID:          rpID,
		RPOrigins:     []string{origin},
	})
}

func (w WebAuthn) List(ctx context.Context, userID int64) ([]WebAuthnCred, error) {
	rows, err := w.DB.QueryContext(ctx, `
		SELECT id, name, aaguid, last_used_at, created_at
		FROM webauthn_credentials WHERE user_id=? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WebAuthnCred
	for rows.Next() {
		var c WebAuthnCred
		var name, aaguid, used, created sql.NullString
		if err := rows.Scan(&c.ID, &name, &aaguid, &used, &created); err != nil {
			return nil, err
		}
		if name.Valid {
			c.Name = &name.String
		}
		if aaguid.Valid {
			c.AAGUID = &aaguid.String
		}
		if used.Valid {
			c.LastUsedAt = &used.String
		}
		if created.Valid {
			c.CreatedAt = &created.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (w WebAuthn) Count(ctx context.Context, userID int64) (int, error) {
	var n int
	err := w.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM webauthn_credentials WHERE user_id=?`, userID).Scan(&n)
	return n, err
}

func (w WebAuthn) BeginRegistration(ctx context.Context, u *User) (token string, options any, err error) {
	n, err := w.Count(ctx, u.ID)
	if err != nil {
		return "", nil, err
	}
	if n >= maxPasskeys {
		return "", nil, fmt.Errorf("limit")
	}
	wa, err := w.engine()
	if err != nil {
		return "", nil, err
	}
	creds, err := w.loadCreds(ctx, u.ID)
	if err != nil {
		return "", nil, err
	}
	creation, session, err := wa.BeginRegistration(waUser{user: u, credentials: creds},
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(descriptors(creds)),
	)
	if err != nil {
		return "", nil, err
	}
	return w.issueChallenge(ctx, &u.ID, WebAuthnRegister, creation.Response, session)
}

func (w WebAuthn) FinishRegistration(ctx context.Context, u *User, token, name string, response json.RawMessage) (*WebAuthnCred, error) {
	session, err := w.consumeChallenge(ctx, token, WebAuthnRegister, u.ID)
	if err != nil || session == nil {
		return nil, fmt.Errorf("challenge")
	}
	wa, err := w.engine()
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		return nil, fmt.Errorf("verify")
	}
	creds, _ := w.loadCreds(ctx, u.ID)
	cred, err := wa.CreateCredential(waUser{user: u, credentials: creds}, *session, parsed)
	if err != nil {
		return nil, fmt.Errorf("verify")
	}
	if name == "" {
		name = "Passkey"
	}
	rec, _ := json.Marshal(cred)
	now := time.Now().UTC().Format(time.RFC3339)
	aaguid := fmt.Sprintf("%x", cred.Authenticator.AAGUID)
	res, err := w.DB.ExecContext(ctx, `
		INSERT INTO webauthn_credentials (user_id, credential_id, name, aaguid, record, counter, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		u.ID, b64(cred.ID), name, aaguid, string(rec), cred.Authenticator.SignCount, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &WebAuthnCred{ID: id, Name: &name, AAGUID: &aaguid, CreatedAt: &now}, nil
}

func (w WebAuthn) BeginLogin(ctx context.Context, u *User) (token string, options any, err error) {
	wa, err := w.engine()
	if err != nil {
		return "", nil, err
	}
	if u == nil {
		assertion, session, err := wa.BeginDiscoverableLogin()
		if err != nil {
			return "", nil, err
		}
		return w.issueChallenge(ctx, nil, WebAuthnLogin, assertion.Response, session)
	}
	creds, err := w.loadCreds(ctx, u.ID)
	if err != nil {
		return "", nil, err
	}
	assertion, session, err := wa.BeginLogin(waUser{user: u, credentials: creds})
	if err != nil {
		return "", nil, err
	}
	return w.issueChallenge(ctx, &u.ID, WebAuthnLogin, assertion.Response, session)
}

func (w WebAuthn) FinishLogin(ctx context.Context, token string, response json.RawMessage) (*User, error) {
	session, userID, err := w.consumeChallengeAny(ctx, token, WebAuthnLogin)
	if err != nil || session == nil {
		return nil, fmt.Errorf("challenge")
	}
	wa, err := w.engine()
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return nil, fmt.Errorf("verify")
	}
	if userID != 0 {
		u, err := Users{DB: w.DB}.ByID(ctx, userID)
		if err != nil || u == nil {
			return nil, fmt.Errorf("verify")
		}
		creds, _ := w.loadCreds(ctx, u.ID)
		if _, err := wa.ValidateLogin(waUser{user: u, credentials: creds}, *session, parsed); err != nil {
			return nil, fmt.Errorf("verify")
		}
		w.touchCred(ctx, parsed.RawID)
		return u, nil
	}
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		u, creds, err := w.userForHandle(ctx, userHandle, rawID)
		if err != nil || u == nil {
			return nil, fmt.Errorf("unknown user")
		}
		return waUser{user: u, credentials: creds}, nil
	}
	cred, err := wa.ValidateDiscoverableLogin(handler, *session, parsed)
	if err != nil {
		return nil, fmt.Errorf("verify")
	}
	u, _, err := w.userForHandle(ctx, parsed.Response.UserHandle, cred.ID)
	if err != nil || u == nil {
		return nil, fmt.Errorf("verify")
	}
	w.touchCred(ctx, parsed.RawID)
	return u, nil
}

func (w WebAuthn) Rename(ctx context.Context, userID, id int64, name string) error {
	res, err := w.DB.ExecContext(ctx, `UPDATE webauthn_credentials SET name=?, updated_at=? WHERE id=? AND user_id=?`,
		name, time.Now().UTC().Format(time.RFC3339), id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (w WebAuthn) Delete(ctx context.Context, userID, id int64) error {
	res, err := w.DB.ExecContext(ctx, `DELETE FROM webauthn_credentials WHERE id=? AND user_id=?`, id, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (w WebAuthn) issueChallenge(ctx context.Context, userID *int64, typ string, options any, session *webauthn.SessionData) (string, any, error) {
	token := RandomString(48)
	now := time.Now().UTC()
	payload := map[string]any{"options": options, "session": session}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", nil, err
	}
	var uid any
	if userID != nil {
		uid = *userID
	}
	_, err = w.DB.ExecContext(ctx, `
		INSERT INTO webauthn_challenges (user_id, token_hash, type, options, expires_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?)`, uid, HashToken(token), typ, string(raw), now.Add(5*time.Minute).Format(time.RFC3339),
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	return token, options, err
}

func (w WebAuthn) consumeChallenge(ctx context.Context, token, typ string, userID int64) (*webauthn.SessionData, error) {
	session, uid, err := w.consumeChallengeAny(ctx, token, typ)
	if err != nil || session == nil {
		return nil, err
	}
	if uid != userID {
		return nil, fmt.Errorf("challenge")
	}
	return session, nil
}

func (w WebAuthn) consumeChallengeAny(ctx context.Context, token, typ string) (*webauthn.SessionData, int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var raw string
	var uid sql.NullInt64
	err := w.DB.QueryRowContext(ctx, `
		SELECT user_id, options FROM webauthn_challenges
		WHERE token_hash=? AND type=? AND consumed_at IS NULL AND expires_at > ?`,
		HashToken(token), typ, now).Scan(&uid, &raw)
	if err != nil {
		return nil, 0, nil
	}
	_, _ = w.DB.ExecContext(ctx, `UPDATE webauthn_challenges SET consumed_at=?, updated_at=? WHERE token_hash=?`, now, now, HashToken(token))
	var wrap struct {
		Session webauthn.SessionData `json:"session"`
	}
	if err := json.Unmarshal([]byte(raw), &wrap); err != nil {
		return nil, 0, err
	}
	return &wrap.Session, uid.Int64, nil
}

func (w WebAuthn) loadCreds(ctx context.Context, userID int64) ([]webauthn.Credential, error) {
	rows, err := w.DB.QueryContext(ctx, `SELECT record FROM webauthn_credentials WHERE user_id=?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []webauthn.Credential
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var c webauthn.Credential
		if json.Unmarshal([]byte(raw), &c) == nil {
			out = append(out, c)
		}
	}
	return out, rows.Err()
}

func (w WebAuthn) userForHandle(ctx context.Context, handle, rawID []byte) (*User, []webauthn.Credential, error) {
	if len(handle) > 0 {
		var id int64
		if _, err := fmt.Sscanf(string(handle), "%d", &id); err == nil && id > 0 {
			u, err := Users{DB: w.DB}.ByID(ctx, id)
			if err != nil || u == nil {
				return nil, nil, err
			}
			creds, err := w.loadCreds(ctx, u.ID)
			return u, creds, err
		}
	}
	if len(rawID) == 0 {
		return nil, nil, nil
	}
	var userID int64
	err := w.DB.QueryRowContext(ctx, `SELECT user_id FROM webauthn_credentials WHERE credential_id=?`, b64(rawID)).Scan(&userID)
	if err != nil {
		return nil, nil, nil
	}
	u, err := Users{DB: w.DB}.ByID(ctx, userID)
	if err != nil || u == nil {
		return u, nil, err
	}
	creds, err := w.loadCreds(ctx, u.ID)
	return u, creds, err
}

func (w WebAuthn) touchCred(ctx context.Context, rawID []byte) {
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = w.DB.ExecContext(ctx, `UPDATE webauthn_credentials SET last_used_at=?, updated_at=? WHERE credential_id=?`,
		now, now, b64(rawID))
}

func descriptors(creds []webauthn.Credential) []protocol.CredentialDescriptor {
	out := make([]protocol.CredentialDescriptor, 0, len(creds))
	for _, c := range creds {
		out = append(out, protocol.CredentialDescriptor{Type: protocol.PublicKeyCredentialType, CredentialID: c.ID})
	}
	return out
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func init() {
	_ = rand.Reader
}
