package httpserver

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/currencydata"
	"savvy-go/internal/firstrun"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
)

// Test roles for createUser: admin is a server admin and admin of the shared
// test space; editor and viewer are plain users with that role in it.
const (
	roleEditor = spaces.RoleEditor
	roleViewer = spaces.RoleViewer
)

type testApp struct {
	t      *testing.T
	db     *sql.DB // the server database
	store  *sqlite.Store
	space  *spaces.Space // the space every createUser member belongs to
	srv    *httptest.Server
	s      *Server
	svc    services
	client *http.Client

	bootstrap *firstrun.Guard // the first-run password of the server
}

// openTestStore opens a store in cfg.DataDir with the embedded migrations.
func openTestStore(t *testing.T, cfg appconfig.Config) *sqlite.Store {
	t.Helper()
	st, err := sqlite.OpenApp(context.Background(), cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// testKeys is a signing key in the test's data directory.
func testKeys(t *testing.T, cfg appconfig.Config) *signing.Holder {
	t.Helper()
	key, err := signing.Load(cfg.DataDir, false)
	if err != nil {
		t.Fatal(err)
	}
	return signing.NewHolder(key)
}

func newTestApp(t *testing.T, opts ...Option) *testApp {
	t.Helper()
	cfg, _ := testConfig(t)
	st := openTestStore(t, cfg)
	guard := firstrun.New(cfg.DataDir, firstrun.Binary)
	opts = append([]Option{WithRates(currencydata.NewOffline()), WithBootstrap(guard)}, opts...)
	s := newServer(cfg, st, testKeys(t, cfg), opts...)
	space, err := servicesOf(s).Spaces.Create(context.Background(), "Test", nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return &testApp{
		t:         t,
		db:        st.SQL(),
		store:     st,
		space:     space,
		srv:       ts,
		s:         s,
		svc:       servicesOf(s),
		client:    ts.Client(),
		bootstrap: guard,
	}
}

// spaceDB is the database of the shared test space.
func (a *testApp) spaceDB() store.DB {
	a.t.Helper()
	d, err := a.store.Space(context.Background(), a.space.ID)
	if err != nil {
		a.t.Fatal(err)
	}
	return d
}

// createUser creates a user. role is a server role (admin, user, guest) or,
// for editor and viewer, a plain user with that role in the shared space; an
// admin also administers the shared space.
func (a *testApp) createUser(email, password, role string, opts ...func(*auth.User)) *auth.User {
	a.t.Helper()
	serverRole, spaceRole := role, ""
	switch role {
	case auth.RoleAdmin:
		spaceRole = spaces.RoleAdmin
	case roleEditor, roleViewer:
		serverRole, spaceRole = auth.RoleUser, role
	}
	u, err := a.svc.Users.Create(context.Background(), "U", email, &password, serverRole)
	if err != nil {
		a.t.Fatal(err)
	}
	if spaceRole != "" {
		if err := a.svc.Spaces.SetMember(context.Background(), a.space.ID, u, spaceRole); err != nil {
			a.t.Fatal(err)
		}
	}
	for _, opt := range opts {
		opt(u)
	}
	return u
}

func (a *testApp) issue(u *auth.User, remember bool) *auth.Issued {
	a.t.Helper()
	issued, err := a.svc.Sessions.Issue(context.Background(), u, "127.0.0.1", "test", remember)
	if err != nil {
		a.t.Fatal(err)
	}
	return issued
}
