package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	repo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/store"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	WebAuthnRegister = "registration"
	WebAuthnLogin    = "authentication"
	maxPasskeys      = 20
)

type WebAuthn struct {
	DB store.DB
	// AppURL is the public URL of the instance, empty when unset.
	AppURL func() string
}

func (w WebAuthn) repo() repo.Repo { return repo.Repo{DB: w.DB} }

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
	LastUsedAt *time.Time
	CreatedAt  *time.Time
}

// ErrNoPublicURL means passkeys cannot work: the relying party is the public
// URL of the instance, and it is not set (or is not a URL).
var ErrNoPublicURL = errors.New("the public URL is not set")

func (w WebAuthn) engine() (*webauthn.WebAuthn, error) {
	u, err := url.Parse(w.AppURL())
	if err != nil || u.Host == "" || u.Hostname() == "" {
		return nil, ErrNoPublicURL
	}
	return webauthn.New(&webauthn.Config{
		RPDisplayName: "Savvy",
		RPID:          u.Hostname(),
		// An origin is scheme and host only, whatever path the instance lives under.
		RPOrigins: []string{u.Scheme + "://" + u.Host},
	})
}

func (w WebAuthn) List(ctx context.Context, userID int64) ([]WebAuthnCred, error) {
	rows, err := w.repo().Passkeys(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]WebAuthnCred, len(rows))
	for i, r := range rows {
		out[i] = WebAuthnCred{ID: r.ID, Name: r.Name, AAGUID: r.AAGUID, LastUsedAt: r.LastUsedAt, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

func (w WebAuthn) Count(ctx context.Context, userID int64) (int, error) {
	return w.repo().CountPasskeys(ctx, userID)
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
	now := time.Now().UTC()
	aaguid := fmt.Sprintf("%x", cred.Authenticator.AAGUID)
	id, err := w.repo().InsertPasskey(ctx, repo.NewPasskey{
		UserID: u.ID, CredentialID: b64(cred.ID), Name: name, AAGUID: aaguid,
		Record: string(rec), Counter: int64(cred.Authenticator.SignCount), Now: now,
	})
	if err != nil {
		return nil, err
	}
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
	ok, err := w.repo().RenamePasskey(ctx, userID, id, name, time.Now().UTC())
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (w WebAuthn) Delete(ctx context.Context, userID, id int64) error {
	ok, err := w.repo().DeletePasskey(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (w WebAuthn) issueChallenge(ctx context.Context, userID *int64, typ string, options any, session *webauthn.SessionData) (string, any, error) {
	token := RandomString(48)
	now := time.Now().UTC()
	raw, err := json.Marshal(map[string]any{"options": options, "session": session})
	if err != nil {
		return "", nil, err
	}
	err = w.repo().InsertPasskeyChallenge(ctx, userID, HashToken(token), typ, string(raw), now.Add(5*time.Minute), now)
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
	row, _ := w.repo().TakePasskeyChallenge(ctx, HashToken(token), typ, time.Now().UTC())
	if row == nil {
		return nil, 0, nil
	}
	var wrap struct {
		Session webauthn.SessionData `json:"session"`
	}
	if err := json.Unmarshal([]byte(row.Options), &wrap); err != nil {
		return nil, 0, err
	}
	return &wrap.Session, row.UserID, nil
}

func (w WebAuthn) loadCreds(ctx context.Context, userID int64) ([]webauthn.Credential, error) {
	recs, err := w.repo().PasskeyRecords(ctx, userID)
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
	userID, err := w.repo().PasskeyUserID(ctx, b64(rawID))
	if err != nil || userID == nil {
		return nil, nil, nil
	}
	u, err := Users{DB: w.DB}.ByID(ctx, *userID)
	if err != nil || u == nil {
		return u, nil, err
	}
	creds, err := w.loadCreds(ctx, u.ID)
	return u, creds, err
}

func (w WebAuthn) touchCred(ctx context.Context, rawID []byte) {
	_ = w.repo().TouchPasskey(ctx, b64(rawID), time.Now().UTC())
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
