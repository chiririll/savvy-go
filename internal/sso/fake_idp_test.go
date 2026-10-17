package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/settings"
)

func ptr[T any](v T) *T { return &v }

// fakeIdP is an OpenID Connect provider: discovery, JWKS, token and userinfo endpoints.
type fakeIdP struct {
	srv  *httptest.Server
	key  *rsa.PrivateKey
	mu   sync.Mutex
	next map[string]any // claims of the ID token the next token request returns
	// userinfo is returned by the userinfo endpoint.
	userinfo map[string]any
	// tokenBody overrides the token response when set.
	tokenBody map[string]any
	// code and verifier the token endpoint saw last.
	lastForm url.Values
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeIdP{key: key}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": f.srv.URL, "authorization_endpoint": f.srv.URL + "/authorize", "token_endpoint": f.srv.URL + "/token",
			"userinfo_endpoint": f.srv.URL + "/userinfo", "jwks_uri": f.srv.URL + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]any{{
			"kty": "RSA", "kid": "k1", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		defer f.mu.Unlock()
		f.lastForm = r.PostForm
		if f.tokenBody != nil {
			_ = json.NewEncoder(w).Encode(f.tokenBody)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "id_token": f.sign(f.next)})
	})
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer at" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(f.userinfo)
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeIdP) sign(claims map[string]any) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims(claims))
	tok.Header["kid"] = "k1"
	s, err := tok.SignedString(f.key)
	if err != nil {
		panic(err)
	}
	return s
}

// idToken are valid claims for the subject, to be adjusted by the test.
func (f *fakeIdP) idToken(sub, email string, verified bool) map[string]any {
	now := time.Now()
	return map[string]any{
		"iss": f.srv.URL, "sub": sub, "aud": "cid", "email": email, "email_verified": verified, "name": "Ann",
		"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
	}
}

type ssoEnv struct {
	ctx context.Context
	svc Service
	idp *fakeIdP
	p   *IdentityProvider
}

func newSSOEnv(t *testing.T, write func(*IdPWrite)) *ssoEnv {
	t.Helper()
	d := dbtest.Server(t)
	e := &ssoEnv{ctx: context.Background(), idp: newFakeIdP(t)}
	e.svc = Service{
		DB: d, Users: auth.Users{DB: d}, Settings: settings.Store{DB: d},
		Cfg: appconfig.Config{}.WithAppURL("https://app.test/"), HTTP: e.idp.srv.Client(),
	}
	in := IdPWrite{
		Name: "Test IdP", Slug: "test", Preset: "custom_oidc", Enabled: ptr(true),
		Fields: map[string]any{
			"discovery_url": e.idp.srv.URL + "/.well-known/openid-configuration",
			"client_id":     "cid", "client_secret": "shh",
		},
	}
	if write != nil {
		write(&in)
	}
	p, err := e.svc.Create(e.ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	e.p = p
	return e
}

// login runs the authorize and callback steps and returns the identity.
func (e *ssoEnv) login(t *testing.T) (NormalizedIdentity, error) {
	t.Helper()
	loc, err := e.svc.AuthorizeURL(e.ctx, *e.p, "")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(loc)
	return e.svc.HandleCallback(e.ctx, *e.p, url.Values{"code": {"abc"}, "state": {u.Query().Get("state")}})
}

func ssoCode(err error) string {
	if e := AsError(err); e != nil {
		return e.Code
	}
	if err != nil {
		return "other: " + err.Error()
	}
	return ""
}
