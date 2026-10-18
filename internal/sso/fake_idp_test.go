package sso

import (
	"context"
	"net/url"
	"testing"

	"savvy-go/internal/appconfig"
	"savvy-go/internal/auth"
	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/settings"
)

func ptr[T any](v T) *T { return &v }

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
