package sso_test

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	authrepo "savvy-go/internal/db/repo/auth"
	ssorepo "savvy-go/internal/db/repo/sso"
)

var ctx = context.Background()

func newRepo(t *testing.T) (ssorepo.Repo, int64) {
	t.Helper()
	d := dbtest.Server(t)
	uid, err := authrepo.Repo{DB: d}.InsertUser(ctx, "Ann", "ann@test.com", "h", "user", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return ssorepo.Repo{DB: d}, uid
}

func provider(slug string, enabled bool) ssorepo.Save {
	return ssorepo.Save{
		Name: slug, Slug: slug, Protocol: "oidc", Preset: "generic", DefaultRole: "user", Enabled: enabled,
		AllowJIT: true, SortOrder: 1, Config: `{"issuer":"x"}`, Secrets: `{}`, ClaimMappings: `{}`, RoleMapping: `[]`,
	}
}

func TestProvidersAreListedFilteredAndUpdated(t *testing.T) {
	r, _ := newRepo(t)
	now := time.Now().UTC()
	a, err := r.Insert(ctx, provider("a", true), now)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := r.Insert(ctx, provider("b", false), now)

	if all, _ := r.List(ctx, ssorepo.Filter{}); len(all) != 2 {
		t.Fatalf("all = %d", len(all))
	}
	if on, _ := r.List(ctx, ssorepo.Filter{EnabledOnly: true}); len(on) != 1 || on[0].ID != a {
		t.Fatalf("enabled = %+v", on)
	}
	if one, _ := r.List(ctx, ssorepo.Filter{ID: b}); len(one) != 1 || one[0].Slug != "b" || one[0].Config != `{"issuer":"x"}` {
		t.Fatalf("by id = %+v", one)
	}
	if one, _ := r.List(ctx, ssorepo.Filter{Slug: "a"}); len(one) != 1 || one[0].ID != a || !one[0].AllowJIT || one[0].CreatedAt == nil {
		t.Fatalf("by slug = %+v", one)
	}
	if n, _ := r.CountEnabled(ctx); n != 1 {
		t.Fatalf("enabled count = %d", n)
	}

	up := provider("a", false)
	up.Name = "Renamed"
	if err := r.Update(ctx, a, up, now); err != nil {
		t.Fatal(err)
	}
	if got, _ := r.List(ctx, ssorepo.Filter{ID: a}); got[0].Name != "Renamed" || got[0].Enabled {
		t.Fatalf("updated = %+v", got[0])
	}
	if err := r.Delete(ctx, a); err != nil {
		t.Fatal(err)
	}
	if all, _ := r.List(ctx, ssorepo.Filter{}); len(all) != 1 {
		t.Fatalf("after delete = %d", len(all))
	}
}

func TestIdentityLinks(t *testing.T) {
	r, user := newRepo(t)
	now := time.Now().UTC()
	p, _ := r.Insert(ctx, provider("a", true), now)
	if l, err := r.LinkBySubject(ctx, p, "sub-1"); l != nil || err != nil {
		t.Fatalf("unknown subject = %v, %v", l, err)
	}
	if err := r.InsertLink(ctx, user, p, "sub-1", `{"email":"a"}`, now); err != nil {
		t.Fatal(err)
	}
	l, _ := r.LinkBySubject(ctx, p, "sub-1")
	if l == nil || l.UserID != user {
		t.Fatalf("link = %+v", l)
	}
	if err := r.TouchLink(ctx, l.ID, `{"email":"b"}`, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if n, _ := r.SSOOnlyOrphans(ctx, p); n != 0 {
		t.Fatalf("orphans = %d for a user who has a password", n)
	}
}

func TestTicketsWorkOnceAndExpire(t *testing.T) {
	r, user := newRepo(t)
	now := time.Now().UTC()
	_ = r.InsertTicket(ctx, "live", user, true, now.Add(2*time.Minute), now)
	_ = r.InsertTicket(ctx, "dead", user, false, now.Add(-time.Minute), now.Add(-time.Hour))

	if _, _, ok, _ := r.TakeTicket(ctx, "dead", now); ok {
		t.Fatal("an expired ticket was accepted")
	}
	uid, needs2FA, ok, err := r.TakeTicket(ctx, "live", now)
	if err != nil || !ok || uid != user || !needs2FA {
		t.Fatalf("take = %d, %v, %v, %v", uid, needs2FA, ok, err)
	}
	if _, _, ok, _ := r.TakeTicket(ctx, "live", now); ok {
		t.Fatal("a ticket must work only once")
	}
	if _, _, ok, _ := r.TakeTicket(ctx, "missing", now); ok {
		t.Fatal("an unknown ticket was accepted")
	}
}

func TestStateIsTakenOnceAndExpires(t *testing.T) {
	r, _ := newRepo(t)
	now := time.Now().UTC()
	p, _ := r.Insert(ctx, provider("a", true), now)
	st := ssorepo.State{ProviderID: p, Nonce: "n", CodeVerifier: "v", RedirectAfter: "/home"}
	_ = r.InsertState(ctx, "live", st, now.Add(10*time.Minute), now)
	_ = r.InsertState(ctx, "dead", st, now.Add(-time.Minute), now.Add(-time.Hour))

	if got, _ := r.TakeState(ctx, "dead", now); got != nil {
		t.Fatal("an expired state was returned")
	}
	if err := r.AttachSAMLRequestID(ctx, "live", "req-1"); err != nil {
		t.Fatal(err)
	}
	got, _ := r.TakeState(ctx, "live", now)
	if got == nil || *got != (ssorepo.State{ProviderID: p, Nonce: "n", CodeVerifier: "v", SAMLRequestID: "req-1", RedirectAfter: "/home"}) {
		t.Fatalf("state = %+v", got)
	}
	if again, _ := r.TakeState(ctx, "live", now); again != nil {
		t.Fatal("a state must be taken only once")
	}
}
