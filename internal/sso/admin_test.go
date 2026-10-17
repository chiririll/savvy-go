package sso

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/settings"
)

// --- provider administration ---

func newAdminEnv(t *testing.T) (Service, context.Context) {
	t.Helper()
	d := dbtest.Server(t)
	return Service{DB: d, Users: auth.Users{DB: d}, Settings: settings.Store{DB: d}}, context.Background()
}

func oidcFields() map[string]any {
	return map[string]any{
		"discovery_url": "https://idp.test/.well-known/openid-configuration",
		"client_id":     "cid",
		"client_secret": "s3cret",
	}
}

func TestCreateValidatesTheProvider(t *testing.T) {
	s, ctx := newAdminEnv(t)
	if _, err := s.Create(ctx, IdPWrite{Slug: "a", Preset: "myspace", Fields: oidcFields()}); ssoCode(
		err,
	) != "unknown_preset" {
		t.Errorf("unknown preset: %q", ssoCode(err))
	}
	for _, slug := range []string{"", "Has Space", "UPPER", "a_b", "../x"} {
		if _, err := s.Create(ctx, IdPWrite{Slug: slug, Preset: "custom_oidc", Fields: oidcFields()}); ssoCode(
			err,
		) != "invalid_slug" {
			t.Errorf("slug %q: %q", slug, ssoCode(err))
		}
	}
	fields := oidcFields()
	delete(fields, "client_id")
	if _, err := s.Create(ctx, IdPWrite{Slug: "a", Preset: "custom_oidc", Fields: fields}); ssoCode(
		err,
	) != "missing_field" {
		t.Errorf("missing client id: %q", ssoCode(err))
	}
	fields = oidcFields()
	fields["client_secret"] = "  "
	if _, err := s.Create(ctx, IdPWrite{Slug: "a", Preset: "custom_oidc", Fields: fields}); ssoCode(
		err,
	) != "missing_field" {
		t.Errorf("blank secret: %q", ssoCode(err))
	}
	if _, err := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "custom_oidc", Fields: oidcFields()}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, IdPWrite{Name: "B", Slug: "a", Preset: "custom_oidc", Fields: oidcFields()}); ssoCode(
		err,
	) != "slug_taken" {
		t.Errorf("duplicate slug: %q", ssoCode(err))
	}
	if all, _ := s.All(ctx); len(all) != 1 {
		t.Errorf("refused providers were stored: %d", len(all))
	}
}

func TestNewProviderStartsDisabledLinksEmail(t *testing.T) {
	s, ctx := newAdminEnv(t)
	p, err := s.Create(
		ctx,
		IdPWrite{Name: "A", Slug: "a", Preset: "google", Fields: map[string]any{"client_id": "cid", "client_secret": "x"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if p.Enabled || !p.AllowJIT || !p.LinkByEmail || p.SyncRoleOnLogin || p.Protocol != "oidc" {
		t.Errorf("defaults %+v", p)
	}
	if s.AnyEnabled(ctx) {
		t.Error("a disabled provider is not offered")
	}
	if p.ClaimMappings["subject"] != "sub" {
		t.Errorf("the preset's claim mappings are the default: %v", p.ClaimMappings)
	}
	if _, err := s.Update(ctx, p.ID, IdPWrite{Enabled: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	if !s.AnyEnabled(ctx) {
		t.Error("an enabled provider is offered")
	}
	if en, _ := s.Enabled(ctx); len(en) != 1 {
		t.Errorf("enabled: %d", len(en))
	}
	if got, _ := s.BySlug(ctx, "a"); got == nil || got.ID != p.ID {
		t.Error("BySlug")
	}
	if got, _ := s.BySlug(ctx, "zz"); got != nil {
		t.Error("BySlug of an unknown slug")
	}
}

func TestSecretNeverLeavesKeptWhenOmitted(t *testing.T) {
	s, ctx := newAdminEnv(t)
	p, err := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "custom_oidc", Fields: oidcFields()})
	if err != nil {
		t.Fatal(err)
	}
	for _, public := range []map[string]any{p.JSON("https://app.test/"), p.PublicJSON()} {
		raw, _ := json.Marshal(public)
		if strings.Contains(string(raw), "s3cret") {
			t.Fatalf("the secret is in %s", raw)
		}
	}
	js := p.JSON("https://app.test/")
	if js["hasClientSecret"] != true || js["callbackUrl"] != "https://app.test/api/auth/sso/a/callback" ||
		js["acsUrl"] != "https://app.test/api/auth/sso/a/acs" {
		t.Errorf("json %v", js)
	}
	if _, ok := p.PublicJSON()["config"]; ok {
		t.Error("the public view shows only what the login page needs")
	}

	edit := oidcFields()
	delete(edit, "client_secret") // the form comes back without it
	edit["client_id"] = "cid2"
	upd, err := s.Update(ctx, p.ID, IdPWrite{Fields: edit})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Secrets["client_secret"] != "s3cret" || upd.Config["client_id"] != "cid2" {
		t.Errorf("secret %v config %v", upd.Secrets, upd.Config)
	}
	edit["client_secret"] = "rotated"
	upd, _ = s.Update(ctx, p.ID, IdPWrite{Fields: edit})
	if upd.Secrets["client_secret"] != "rotated" {
		t.Errorf("a new secret replaces the old: %v", upd.Secrets)
	}
}

func TestUpdateKeepsWhatIsNotSent(t *testing.T) {
	s, ctx := newAdminEnv(t)
	p, _ := s.Create(
		ctx,
		IdPWrite{
			Name:            "A",
			Slug:            "a",
			Preset:          "custom_oidc",
			Fields:          oidcFields(),
			DefaultRole:     "guest",
			SyncRoleOnLogin: ptr(true),
			SortOrder:       ptr(3),
		},
	)
	upd, err := s.Update(ctx, p.ID, IdPWrite{Name: "Renamed"})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Name != "Renamed" || upd.DefaultRole != "guest" || !upd.SyncRoleOnLogin || upd.SortOrder != 3 ||
		upd.Config["client_id"] != "cid" ||
		upd.Secrets["client_secret"] != "s3cret" {
		t.Errorf("updated %+v", upd)
	}
	if got, err := s.Update(ctx, 999, IdPWrite{Name: "x"}); got != nil || err != nil {
		t.Errorf("unknown provider: %v, %v", got, err)
	}
	other, _ := s.Create(ctx, IdPWrite{Name: "B", Slug: "b", Preset: "custom_oidc", Fields: oidcFields()})
	if _, err := s.Update(ctx, other.ID, IdPWrite{Slug: "a"}); ssoCode(err) != "slug_taken" {
		t.Errorf("taking another slug: %q", ssoCode(err))
	}
	if _, err := s.Update(ctx, other.ID, IdPWrite{Slug: "b"}); err != nil {
		t.Errorf("keeping its own slug: %v", err)
	}
	if _, err := s.Update(ctx, other.ID, IdPWrite{Slug: "Not Valid"}); ssoCode(err) != "invalid_slug" {
		t.Errorf("invalid slug: %q", ssoCode(err))
	}
}

func TestDeleteRefusesStrandSSOOnlyUsers(t *testing.T) {
	s, ctx := newAdminEnv(t)
	p, _ := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "custom_oidc", Fields: oidcFields(), Enabled: ptr(true)})
	if _, err := s.Users.Create(ctx, "Admin", "root@test.com", ptr("pw-pw-pw"), auth.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Provision(ctx, *p, ident("sub", "jit@test.com", true)); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, p.ID); err == nil {
		t.Fatal("the only way in for an SSO-only user was removed")
	}
	if got, _ := s.ByID(ctx, p.ID); got == nil {
		t.Error("the provider is gone")
	}

	empty, _ := s.Create(ctx, IdPWrite{Name: "B", Slug: "b", Preset: "custom_oidc", Fields: oidcFields()})
	if err := s.Delete(ctx, empty.ID); err != nil {
		t.Errorf("a provider nobody depends on: %v", err)
	}
}

func TestTicketsStatesAreSingleUse(t *testing.T) {
	s, ctx := newAdminEnv(t)
	u, _ := s.Users.Create(ctx, "A", "a@test.com", ptr("pw-pw-pw"), "")

	ticket, err := s.IssueTicket(ctx, u.ID, true)
	if err != nil || len(ticket) != 64 {
		t.Fatalf("ticket %q, %v", ticket, err)
	}
	id, needs2FA, ok := s.ConsumeTicket(ctx, ticket)
	if !ok || id != u.ID || !needs2FA {
		t.Errorf("consume: %d %v %v", id, needs2FA, ok)
	}
	if _, _, ok := s.ConsumeTicket(ctx, ticket); ok {
		t.Error("a ticket worked twice")
	}
	if _, _, ok := s.ConsumeTicket(ctx, "forged"); ok {
		t.Error("an invented ticket worked")
	}

	p, err := s.Create(ctx, IdPWrite{Name: "A", Slug: "a", Preset: "custom_oidc", Fields: oidcFields()})
	if err != nil {
		t.Fatal(err)
	}
	state, err := s.SaveState(ctx, IdPState{ProviderID: p.ID, SAMLRequestID: "r"})
	if err != nil || len(state) != 48 {
		t.Fatalf("state %q, %v", state, err)
	}
	if err := s.AttachSAMLRequestID(ctx, state, "req-77"); err != nil {
		t.Fatal(err)
	}
	st, _ := s.ConsumeState(ctx, state)
	if st == nil || st.SAMLRequestID != "req-77" || st.ProviderID != p.ID {
		t.Errorf("state %+v", st)
	}
	if again, _ := s.ConsumeState(ctx, state); again != nil {
		t.Error("a state worked twice")
	}
}

func TestCallbackURLsFollowPublicURL(t *testing.T) {
	s := Service{}
	p := IdentityProvider{Slug: "x"}
	if got := s.CallbackURL(context.Background(), p); got != "/api/auth/sso/x/callback" {
		t.Errorf("without a public URL: %q", got)
	}
}
