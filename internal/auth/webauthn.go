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
	"github.com/chiririll/savvy-plus/internal/db"
	"github.com/chiririll/savvy-plus/internal/db/sqlc"
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
	rows, err := db.Q(w.DB).ListWebAuthnCredentials(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]WebAuthnCred, 0, len(rows))
	for _, r := range rows {
		c := WebAuthnCred{ID: r.ID}
		if r.Name.Valid {
			c.Name = &r.Name.String
		}
		if r.Aaguid.Valid {
			c.AAGUID = &r.Aaguid.String
		}
		if r.LastUsedAt.Valid {
			c.LastUsedAt = &r.LastUsedAt.String
		}
		if r.CreatedAt.Valid {
			c.CreatedAt = &r.CreatedAt.String
		}
		out = append(out, c)
	}
	return out, nil
}

func (w WebAuthn) Count(ctx context.Context, userID int64) (int, error) {
	n, err := db.Q(w.DB).CountWebAuthnCredentials(ctx, userID)
	return int(n), err
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
	res, err := db.Q(w.DB).InsertWebAuthnCredential(ctx, sqlc.InsertWebAuthnCredentialParams{
		UserID: u.ID, CredentialID: b64(cred.ID), Name: db.NS(name), Aaguid: db.NS(aaguid),
		Record: string(rec), Counter: int64(cred.Authenticator.SignCount), CreatedAt: db.NS(now), UpdatedAt: db.NS(now),
	})
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
	res, err := db.Q(w.DB).RenameWebAuthnCredential(ctx, sqlc.RenameWebAuthnCredentialParams{
		Name: db.NS(name), UpdatedAt: db.NS(time.Now().UTC().Format(time.RFC3339)), ID: id, UserID: userID,
	})
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
	res, err := db.Q(w.DB).DeleteWebAuthnCredential(ctx, sqlc.DeleteWebAuthnCredentialParams{ID: id, UserID: userID})
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
	err = db.Q(w.DB).InsertWebAuthnChallenge(ctx, sqlc.InsertWebAuthnChallengeParams{
		UserID: db.NullInt64(userID), TokenHash: HashToken(token), Type: typ, Options: string(raw),
		ExpiresAt: now.Add(5 * time.Minute).Format(time.RFC3339),
		CreatedAt: db.NS(now.Format(time.RFC3339)), UpdatedAt: db.NS(now.Format(time.RFC3339)),
	})
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
	row, err := db.Q(w.DB).GetOpenWebAuthnChallenge(ctx, sqlc.GetOpenWebAuthnChallengeParams{
		TokenHash: HashToken(token), Type: typ, ExpiresAt: now,
	})
	if err != nil {
		return nil, 0, nil
	}
	_ = db.Q(w.DB).ConsumeWebAuthnChallenge(ctx, sqlc.ConsumeWebAuthnChallengeParams{
		ConsumedAt: db.NS(now), UpdatedAt: db.NS(now), TokenHash: HashToken(token),
	})
	raw := row.Options
	uid := row.UserID
	var wrap struct {
		Session webauthn.SessionData `json:"session"`
	}
	if err := json.Unmarshal([]byte(raw), &wrap); err != nil {
		return nil, 0, err
	}
	return &wrap.Session, uid.Int64, nil
}

func (w WebAuthn) loadCreds(ctx context.Context, userID int64) ([]webauthn.Credential, error) {
	recs, err := db.Q(w.DB).ListWebAuthnRecords(ctx, userID)
	if err != nil {
		return nil, err
	}
	var out []webauthn.Credential
	for _, raw := range recs {
		var c webauthn.Credential
		if json.Unmarshal([]byte(raw), &c) == nil {
			out = append(out, c)
		}
	}
	return out, nil
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
	userID, err := db.Q(w.DB).GetWebAuthnUserIDByCredential(ctx, b64(rawID))
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
	_ = db.Q(w.DB).TouchWebAuthnCredential(ctx, sqlc.TouchWebAuthnCredentialParams{
		LastUsedAt: db.NS(now), UpdatedAt: db.NS(now), CredentialID: b64(rawID),
	})
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
