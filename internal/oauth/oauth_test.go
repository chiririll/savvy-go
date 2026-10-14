package oauth_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/auth"
	authrepo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/oauth"
	"savvy-go/internal/store/sqlite"
)

const (
	redirect = "https://claude.ai/api/mcp/auth_callback"
	vOK      = "vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv"
	vBad     = "wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww"
)

type env struct {
	svc  *oauth.Service
	user *auth.User
	now  time.Time
	tok  auth.APITokens
}

func newEnv(t *testing.T) *env {
	t.Helper()
	st, err := sqlite.OpenApp(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	srv := st.Server()
	uid, err := authrepo.Repo{DB: srv}.InsertUser(context.Background(), "Ann", "ann@test.com", "hash", "user", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	u, err := auth.Users{DB: srv}.ByID(context.Background(), uid)
	if err != nil || u == nil {
		t.Fatalf("user: %v %v", u, err)
	}
	e := &env{user: u, now: time.Now().UTC(), tok: auth.APITokens{DB: srv}}
	e.svc = oauth.New(srv)
	e.svc.Now = func() time.Time { return e.now }
	return e
}

// reg registers an app the way an admin-opened window allows: one at a time.
func (e *env) reg(ctx context.Context, name string, uris []string) (*oauth.Client, error) {
	e.svc.OpenRegistration(oauth.RegistrationWindow)
	return e.svc.Register(ctx, name, uris, oauth.RequestInfo{})
}

func challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (e *env) client(t *testing.T) string {
	t.Helper()
	c, err := e.reg(context.Background(), "Claude", []string{redirect})
	if err != nil {
		t.Fatal(err)
	}
	return c.ID
}

func (e *env) code(t *testing.T, clientID, verifier string) string {
	t.Helper()
	code, err := e.svc.IssueCode(context.Background(), oauth.CodeRequest{
		ClientID: clientID, RedirectURI: redirect, CodeChallenge: challenge(verifier), CodeChallengeMethod: "S256", User: e.user, AllSpaces: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	var oe *oauth.Error
	if !errors.As(err, &oe) || oe.Code != code {
		t.Fatalf("err = %v, want oauth error %q", err, code)
	}
}

func TestRegisterValidatesRedirectURIs(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	for _, uris := range [][]string{
		nil,
		{"http://evil.example/cb"},
		{"https://x.example/cb#frag"},
		{"javascript:alert(1)"},
		{"/relative"},
		{redirect, "http://evil.example/cb"},
	} {
		_, err := e.reg(ctx, "X", uris)
		wantCode(t, err, "invalid_redirect_uri")
	}
	for _, ok := range []string{"http://localhost:6274/cb", "http://127.0.0.1/cb", "http://[::1]:9/cb", redirect} {
		if _, err := e.reg(ctx, "X", []string{ok}); err != nil {
			t.Errorf("%s rejected: %v", ok, err)
		}
	}
}

func TestRegisterDefaultsAndBoundsTheName(t *testing.T) {
	e := newEnv(t)
	c, err := e.reg(context.Background(), "  ", []string{redirect})
	if err != nil || c.Name == "" {
		t.Fatalf("blank name: %+v %v", c, err)
	}
	c, err = e.reg(context.Background(), strings.Repeat("n", 500), []string{redirect})
	if err != nil || len(c.Name) > 100 {
		t.Fatalf("long name kept: %d %v", len(c.Name), err)
	}
}

func TestRegisterIsCapped(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	for i := 0; i < oauth.MaxClients; i++ {
		if _, err := e.reg(ctx, "X", []string{redirect}); err != nil {
			t.Fatalf("client %d: %v", i, err)
		}
	}
	_, err := e.reg(ctx, "X", []string{redirect})
	wantCode(t, err, "server_error")
	// Idle clients age out, which frees room again.
	e.now = e.now.Add(48 * time.Hour)
	if _, err := e.reg(ctx, "X", []string{redirect}); err != nil {
		t.Fatalf("after aging: %v", err)
	}
}

func TestIssueCodeChecksClientRedirectAndPKCE(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	id := e.client(t)
	base := oauth.CodeRequest{ClientID: id, RedirectURI: redirect, CodeChallenge: challenge(vOK), CodeChallengeMethod: "S256", User: e.user, AllSpaces: true}

	bad := base
	bad.ClientID = "nope"
	_, err := e.svc.IssueCode(ctx, bad)
	wantCode(t, err, "invalid_client")

	bad = base
	bad.RedirectURI = "https://evil.example/cb"
	_, err = e.svc.IssueCode(ctx, bad)
	wantCode(t, err, "invalid_request")

	bad = base
	bad.CodeChallenge = ""
	_, err = e.svc.IssueCode(ctx, bad)
	wantCode(t, err, "invalid_request")

	bad = base
	bad.CodeChallengeMethod = "plain"
	_, err = e.svc.IssueCode(ctx, bad)
	wantCode(t, err, "invalid_request")
}

func TestExchangeIssuesAReadOnlyShortLivedAPIToken(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	id := e.client(t)
	code := e.code(t, id, vOK)

	ts, err := e.svc.Exchange(ctx, id, code, vOK, redirect)
	if err != nil {
		t.Fatal(err)
	}
	if ts.ExpiresIn <= 0 || ts.ExpiresIn > 3600 || ts.RefreshToken == "" || ts.AccessToken == "" {
		t.Fatalf("token set = %+v", ts)
	}
	tok, err := e.tok.Resolve(ctx, ts.AccessToken)
	if err != nil || tok == nil {
		t.Fatalf("access token does not resolve: %v %v", tok, err)
	}
	if tok.Allows("transactions", true) || !tok.Allows("transactions", false) || tok.Allows("transfers", false) || !tok.AllSpaces || tok.UserID != e.user.ID || tok.ExpiresAt == nil || !strings.Contains(tok.Name, "Claude") {
		t.Fatalf("token = %+v", tok)
	}
}

func TestCodeIsSingleUseEvenWhenTheFirstTryFails(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	id := e.client(t)
	code := e.code(t, id, vOK)
	_, err := e.svc.Exchange(ctx, id, code, vBad, redirect)
	wantCode(t, err, "invalid_grant")
	_, err = e.svc.Exchange(ctx, id, code, vOK, redirect)
	wantCode(t, err, "invalid_grant")
}

func TestExchangeBindsCodeToClientAndRedirect(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	id := e.client(t)
	other := e.client(t)

	_, err := e.svc.Exchange(ctx, other, e.code(t, id, vOK), vOK, redirect)
	wantCode(t, err, "invalid_grant")
	_, err = e.svc.Exchange(ctx, id, e.code(t, id, vOK), vOK, "https://claude.ai/other")
	wantCode(t, err, "invalid_grant")
}

func TestCodeExpires(t *testing.T) {
	e := newEnv(t)
	id := e.client(t)
	code := e.code(t, id, vOK)
	e.now = e.now.Add(2 * time.Minute)
	_, err := e.svc.Exchange(context.Background(), id, code, vOK, redirect)
	wantCode(t, err, "invalid_grant")
}

func TestRefreshRotatesAndRetiresTheOldTokens(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	id := e.client(t)
	first, err := e.svc.Exchange(ctx, id, e.code(t, id, vOK), vOK, redirect)
	if err != nil {
		t.Fatal(err)
	}
	e.now = e.now.Add(10 * time.Minute)
	second, err := e.svc.Refresh(ctx, id, first.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if second.AccessToken == first.AccessToken || second.RefreshToken == first.RefreshToken {
		t.Fatal("refresh must rotate both tokens")
	}
	if tok, _ := e.tok.Resolve(ctx, first.AccessToken); tok != nil {
		t.Fatal("the previous access token must stop working")
	}
	if tok, _ := e.tok.Resolve(ctx, second.AccessToken); tok == nil {
		t.Fatal("the new access token must work")
	}
	_, err = e.svc.Refresh(ctx, id, first.RefreshToken)
	wantCode(t, err, "invalid_grant")
	if _, err := e.svc.Refresh(ctx, id, second.RefreshToken); err != nil {
		t.Fatalf("the newest refresh token must still work: %v", err)
	}
}

func TestRefreshRejectsOtherClientsExpiredAndRevoked(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	id := e.client(t)
	other := e.client(t)
	ts, _ := e.svc.Exchange(ctx, id, e.code(t, id, vOK), vOK, redirect)

	_, err := e.svc.Refresh(ctx, other, ts.RefreshToken)
	wantCode(t, err, "invalid_grant")

	// Revoking the connection in the API tokens list ends the refresh token too.
	toks, _ := e.tok.List(ctx, e.user.ID)
	if len(toks) != 1 {
		t.Fatalf("tokens = %d", len(toks))
	}
	if err := e.tok.Revoke(ctx, e.user.ID, toks[0].ID); err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.Refresh(ctx, id, ts.RefreshToken)
	wantCode(t, err, "invalid_grant")

	ts2, _ := e.svc.Exchange(ctx, id, e.code(t, id, vOK), vOK, redirect)
	e.now = e.now.Add(oauth.RefreshTTL + time.Hour)
	_, err = e.svc.Refresh(ctx, id, ts2.RefreshToken)
	wantCode(t, err, "invalid_grant")
}

func TestRegistrationIsClosedUnlessAnAdminOpenedIt(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	e.svc.CloseRegistration()
	_, err := e.svc.Register(ctx, "X", []string{redirect}, oauth.RequestInfo{})
	wantCode(t, err, "access_denied")
	if _, open := e.svc.RegistrationOpenUntil(); open {
		t.Fatal("registration reports open after close")
	}

	until := e.svc.OpenRegistration(oauth.RegistrationWindow)
	if !until.Equal(e.now.Add(5 * time.Minute)) {
		t.Fatalf("window ends %v, want 5 minutes from now", until)
	}
	if got, open := e.svc.RegistrationOpenUntil(); !open || !got.Equal(until) {
		t.Fatalf("open = %v until %v", open, got)
	}
	e.now = e.now.Add(5*time.Minute + time.Second)
	_, err = e.svc.Register(ctx, "X", []string{redirect}, oauth.RequestInfo{})
	wantCode(t, err, "access_denied")
	if _, open := e.svc.RegistrationOpenUntil(); open {
		t.Fatal("registration still open after the window")
	}
}

func TestRegisteredClientsKeepWorkingAfterTheWindowCloses(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	id := e.client(t)
	ts, err := e.svc.Exchange(ctx, id, e.code(t, id, vOK), vOK, redirect)
	if err != nil {
		t.Fatal(err)
	}
	e.svc.CloseRegistration()
	if _, err := e.svc.Refresh(ctx, id, ts.RefreshToken); err != nil {
		t.Fatalf("an existing connection must survive a closed window: %v", err)
	}
}

func TestTheFirstRegistrationClosesTheWindow(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	e.svc.OpenRegistration(oauth.RegistrationWindow)

	// A rejected attempt does not use the window up.
	_, err := e.svc.Register(ctx, "Bad", []string{"http://evil.example/cb"}, oauth.RequestInfo{})
	wantCode(t, err, "invalid_redirect_uri")
	if _, open := e.svc.RegistrationOpenUntil(); !open {
		t.Fatal("a failed registration closed the window")
	}

	if _, err := e.svc.Register(ctx, "Claude", []string{redirect}, oauth.RequestInfo{}); err != nil {
		t.Fatal(err)
	}
	if _, open := e.svc.RegistrationOpenUntil(); open {
		t.Fatal("the window must close after one registration")
	}
	_, err = e.svc.Register(ctx, "Second", []string{redirect}, oauth.RequestInfo{})
	wantCode(t, err, "access_denied")
}

func TestRegisterKeepsAndBoundsTheRequestInfo(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	e.svc.OpenRegistration(oauth.RegistrationWindow)
	c, err := e.svc.Register(ctx, "Claude", []string{redirect}, oauth.RequestInfo{IP: "203.0.113.7", UserAgent: strings.Repeat("u", 1000), Origin: "https://claude.ai"})
	if err != nil {
		t.Fatal(err)
	}
	list, err := e.svc.Clients(ctx)
	if err != nil || len(list) != 1 || list[0].ID != c.ID {
		t.Fatalf("clients = %+v, %v", list, err)
	}
	got := list[0]
	if got.Request.IP != "203.0.113.7" || got.Request.Origin != "https://claude.ai" || len(got.Request.UserAgent) != 300 || got.CreatedAt.IsZero() {
		t.Fatalf("client = %+v", got)
	}
}

func TestRemoveRevokesTheAppsTokens(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()
	id := e.client(t)
	ts, _ := e.svc.Exchange(ctx, id, e.code(t, id, vOK), vOK, redirect)
	if ok, err := e.svc.Remove(ctx, id, e.user); !ok || err != nil {
		t.Fatalf("remove = %v, %v", ok, err)
	}
	if tok, _ := e.tok.Resolve(ctx, ts.AccessToken); tok != nil {
		t.Fatal("access token outlived its app")
	}
	_, err := e.svc.Refresh(ctx, id, ts.RefreshToken)
	wantCode(t, err, "invalid_grant")
	if ok, _ := e.svc.Remove(ctx, id, e.user); ok {
		t.Fatal("removing twice must report false")
	}
}
