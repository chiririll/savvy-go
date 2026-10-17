package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/fxamacker/cbor/v2"

	"savvy-go/internal/appconfig"
)

const (
	waOrigin = "https://savvy.test"
	waRPID   = "savvy.test"

	flagUserPresent  = 0x01
	flagUserVerified = 0x04
	flagAttested     = 0x40
)

// authenticator is a software passkey: it makes the same responses a browser
// relays from a device, with "none" attestation.
type authenticator struct {
	t       *testing.T
	key     *ecdsa.PrivateKey
	credID  []byte
	counter uint32
	// noUV makes assertions without the user-verified flag: a key with no PIN.
	noUV bool
}

func newAuthenticator(t *testing.T) *authenticator {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		t.Fatal(err)
	}
	return &authenticator{t: t, key: key, credID: id}
}

func b64u(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// challengeOf reads the challenge out of the options the server handed to the browser.
func challengeOf(t *testing.T, options any) string {
	t.Helper()
	raw, err := json.Marshal(options)
	if err != nil {
		t.Fatal(err)
	}
	var o struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(raw, &o); err != nil || o.Challenge == "" {
		t.Fatalf("options without a challenge: %s", raw)
	}
	return o.Challenge
}

func clientData(t *testing.T, typ, challenge, origin string) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]string{"type": typ, "challenge": challenge, "origin": origin})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (a *authenticator) authData(rpID string, flags byte) []byte {
	hash := sha256.Sum256([]byte(rpID))
	out := append([]byte{}, hash[:]...)
	out = append(out, flags)
	return binary.BigEndian.AppendUint32(out, a.counter)
}

// register answers the creation options.
func (a *authenticator) register(options any) json.RawMessage {
	a.t.Helper()
	cose, err := cbor.Marshal(map[int]any{
		1: 2, 3: -7, -1: 1,
		-2: a.key.PublicKey.X.FillBytes(make([]byte, 32)),
		-3: a.key.PublicKey.Y.FillBytes(make([]byte, 32)),
	})
	if err != nil {
		a.t.Fatal(err)
	}
	data := a.authData(waRPID, flagUserPresent|flagUserVerified|flagAttested)
	data = append(data, make([]byte, 16)...) // AAGUID
	data = binary.BigEndian.AppendUint16(data, uint16(len(a.credID)))
	data = append(data, a.credID...)
	data = append(data, cose...)
	att, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": data})
	if err != nil {
		a.t.Fatal(err)
	}
	return a.response(map[string]string{
		"clientDataJSON":    b64u(clientData(a.t, "webauthn.create", challengeOf(a.t, options), waOrigin)),
		"attestationObject": b64u(att),
	})
}

// assert answers the request options; userHandle is what a discoverable login carries.
func (a *authenticator) assert(options any, userHandle []byte) json.RawMessage {
	return a.assertWith(options, userHandle, waOrigin, a.key)
}

func (a *authenticator) assertWith(
	options any,
	userHandle []byte,
	origin string,
	key *ecdsa.PrivateKey,
) json.RawMessage {
	a.t.Helper()
	a.counter++
	cd := clientData(a.t, "webauthn.get", challengeOf(a.t, options), origin)
	flags := byte(flagUserPresent | flagUserVerified)
	if a.noUV {
		flags = flagUserPresent
	}
	data := a.authData(waRPID, flags)
	hash := sha256.Sum256(cd)
	digest := sha256.Sum256(append(append([]byte{}, data...), hash[:]...))
	sig, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	if err != nil {
		a.t.Fatal(err)
	}
	return a.response(map[string]string{
		"clientDataJSON":    b64u(cd),
		"authenticatorData": b64u(data),
		"signature":         b64u(sig),
		"userHandle":        b64u(userHandle),
	})
}

func (a *authenticator) response(inner map[string]string) json.RawMessage {
	a.t.Helper()
	b, err := json.Marshal(map[string]any{
		"id": b64u(a.credID), "rawId": b64u(a.credID), "type": "public-key", "response": inner,
	})
	if err != nil {
		a.t.Fatal(err)
	}
	return b
}

func newWebAuthn(t *testing.T) (WebAuthn, *User, *authenticator) {
	t.Helper()
	d := newDB(t)
	w := WebAuthn{DB: d, Cfg: appconfig.Config{}.WithAppURL(waOrigin)}
	u := newUser(t, d, "ann@savvy.test", "user")
	dev := newAuthenticator(t)
	token, options, err := w.BeginRegistration(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.FinishRegistration(context.Background(), u, token, "Laptop", dev.register(options)); err != nil {
		t.Fatalf("registering the passkey: %v", err)
	}
	return w, u, dev
}

func TestPasskeyRegistrationStoresTheCredential(t *testing.T) {
	w, u, _ := newWebAuthn(t)
	creds, err := w.List(context.Background(), u.ID)
	if err != nil || len(creds) != 1 || creds[0].Name == nil || *creds[0].Name != "Laptop" {
		t.Fatalf("credentials %+v, %v", creds, err)
	}
}

func TestPasskeyLoginForAKnownUser(t *testing.T) {
	ctx := context.Background()
	w, u, dev := newWebAuthn(t)

	token, options, err := w.BeginLogin(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	got, err := w.FinishLogin(ctx, token, dev.assert(options, nil))
	if err != nil || got == nil || got.ID != u.ID {
		t.Fatalf("login = %+v, %v", got, err)
	}
	creds, _ := w.List(ctx, u.ID)
	if creds[0].LastUsedAt == nil {
		t.Error("a login marks the passkey as used")
	}

	if _, err := w.FinishLogin(ctx, token, dev.assert(options, nil)); err == nil {
		t.Error("a challenge works once")
	}
}

func TestPasskeyDiscoverableLoginFindsTheUser(t *testing.T) {
	ctx := context.Background()
	w, u, dev := newWebAuthn(t)

	token, options, err := w.BeginLogin(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := w.FinishLogin(ctx, token, dev.assert(options, []byte(strconv.FormatInt(u.ID, 10))))
	if err != nil || got == nil || got.ID != u.ID {
		t.Fatalf("login = %+v, %v", got, err)
	}
}

func TestPasskeyLoginNeedsUserVerification(t *testing.T) {
	ctx := context.Background()
	w, u, dev := newWebAuthn(t)
	handle := []byte(strconv.FormatInt(u.ID, 10))

	dev.noUV = true
	token, options, _ := w.BeginLogin(ctx, u)
	if got, err := w.FinishLogin(ctx, token, dev.assert(options, nil)); err == nil || got != nil {
		t.Errorf("a tap without a PIN signed in (user-bound): %+v", got)
	}
	token, options, _ = w.BeginLogin(ctx, nil)
	if got, err := w.FinishLogin(ctx, token, dev.assert(options, handle)); err == nil || got != nil {
		t.Errorf("a tap without a PIN signed in (discoverable): %+v", got)
	}

	dev.noUV = false
	token, options, _ = w.BeginLogin(ctx, u)
	if got, err := w.FinishLogin(ctx, token, dev.assert(options, nil)); err != nil || got == nil {
		t.Errorf("a verified assertion is refused (user-bound): %v", err)
	}
	token, options, _ = w.BeginLogin(ctx, nil)
	if got, err := w.FinishLogin(ctx, token, dev.assert(options, handle)); err != nil || got == nil {
		t.Errorf("a verified assertion is refused (discoverable): %v", err)
	}
}

func TestPasskeyOptionsAskForUserVerification(t *testing.T) {
	ctx := context.Background()
	w, u, _ := newWebAuthn(t)

	_, reg, err := w.BeginRegistration(ctx, u)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(reg)
	var r struct {
		AuthenticatorSelection struct {
			UserVerification string `json:"userVerification"`
			ResidentKey      string `json:"residentKey"`
		} `json:"authenticatorSelection"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	if r.AuthenticatorSelection.UserVerification != "required" || r.AuthenticatorSelection.ResidentKey != "required" {
		t.Errorf("registration selection = %+v, want userVerification and residentKey required", r.AuthenticatorSelection)
	}

	for name, user := range map[string]*User{"user-bound": u, "discoverable": nil} {
		_, opts, err := w.BeginLogin(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(opts)
		var l struct {
			UserVerification string `json:"userVerification"`
		}
		_ = json.Unmarshal(raw, &l)
		if l.UserVerification != "required" {
			t.Errorf("%s login userVerification = %q, want required", name, l.UserVerification)
		}
	}
}

func storedSignCount(t *testing.T, w WebAuthn, userID int64) uint32 {
	t.Helper()
	creds, err := w.loadCreds(context.Background(), userID)
	if err != nil || len(creds) != 1 {
		t.Fatalf("credentials %d, %v", len(creds), err)
	}
	return creds[0].Authenticator.SignCount
}

func TestPasskeyLoginMovesTheSignCountForward(t *testing.T) {
	ctx := context.Background()
	w, u, dev := newWebAuthn(t)
	handle := []byte(strconv.FormatInt(u.ID, 10))

	token, options, _ := w.BeginLogin(ctx, u)
	if _, err := w.FinishLogin(ctx, token, dev.assert(options, nil)); err != nil {
		t.Fatal(err)
	}
	if got := storedSignCount(t, w, u.ID); got != dev.counter {
		t.Errorf("stored count %d after a user-bound login, device at %d", got, dev.counter)
	}

	token, options, _ = w.BeginLogin(ctx, nil)
	if _, err := w.FinishLogin(ctx, token, dev.assert(options, handle)); err != nil {
		t.Fatal(err)
	}
	if got := storedSignCount(t, w, u.ID); got != dev.counter {
		t.Errorf("stored count %d after a discoverable login, device at %d", got, dev.counter)
	}
}

func TestPasskeyLoginRefusesAClonedAuthenticator(t *testing.T) {
	ctx := context.Background()
	w, u, dev := newWebAuthn(t)
	handle := []byte(strconv.FormatInt(u.ID, 10))

	token, options, _ := w.BeginLogin(ctx, u)
	if _, err := w.FinishLogin(ctx, token, dev.assert(options, nil)); err != nil {
		t.Fatal(err)
	}
	token, options, _ = w.BeginLogin(ctx, u)
	if _, err := w.FinishLogin(ctx, token, dev.assert(options, nil)); err != nil {
		t.Fatal(err)
	}
	stored := storedSignCount(t, w, u.ID)

	// The clone's counter is behind what the real device already used.
	dev.counter = 0
	token, options, _ = w.BeginLogin(ctx, u)
	if got, err := w.FinishLogin(ctx, token, dev.assert(options, nil)); err == nil || got != nil {
		t.Errorf("a stale counter signed in (user-bound): %+v", got)
	}
	dev.counter = 0
	token, options, _ = w.BeginLogin(ctx, nil)
	if got, err := w.FinishLogin(ctx, token, dev.assert(options, handle)); err == nil || got != nil {
		t.Errorf("a stale counter signed in (discoverable): %+v", got)
	}
	if got := storedSignCount(t, w, u.ID); got != stored {
		t.Errorf("a refused login changed the stored count from %d to %d", stored, got)
	}

	// The real device, ahead of the stored count, still signs in.
	dev.counter = stored
	token, options, _ = w.BeginLogin(ctx, u)
	if _, err := w.FinishLogin(ctx, token, dev.assert(options, nil)); err != nil {
		t.Errorf("the real device is refused: %v", err)
	}
}

func TestPasskeyLoginRefusesWhatTheDeviceDidNotSign(t *testing.T) {
	ctx := context.Background()
	w, u, dev := newWebAuthn(t)
	stranger, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		resp func(options any) json.RawMessage
	}{
		{"signed by another key", func(o any) json.RawMessage { return dev.assertWith(o, nil, waOrigin, stranger) }},
		{
			"another site's origin",
			func(o any) json.RawMessage { return dev.assertWith(o, nil, "https://evil.test", dev.key) },
		},
		{"not a response", func(any) json.RawMessage { return json.RawMessage(`{}`) }},
	}
	for _, tc := range tests {
		token, options, err := w.BeginLogin(ctx, u)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := w.FinishLogin(ctx, token, tc.resp(options)); err == nil || got != nil {
			t.Errorf("%s: logged in as %+v", tc.name, got)
		}
	}

	// An answer to one challenge does not open another.
	_, first, _ := w.BeginLogin(ctx, u)
	token, _, _ := w.BeginLogin(ctx, u)
	if got, err := w.FinishLogin(ctx, token, dev.assert(first, nil)); err == nil || got != nil {
		t.Errorf("a response to another challenge: %+v", got)
	}

	if got, err := w.FinishLogin(ctx, "no-such-token", dev.assert(first, nil)); err == nil || got != nil {
		t.Errorf("an unknown challenge: %+v", got)
	}
}
