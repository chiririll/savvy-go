package spaces_test

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	authrepo "savvy-go/internal/db/repo/auth"
	spacesrepo "savvy-go/internal/db/repo/spaces"
	"savvy-go/internal/store"
)

var ctx = context.Background()

// registry is a server database with two users and one space owned by the first.
type registry struct {
	db       store.DB
	repo     spacesrepo.Repo
	ann, bob int64
	space    int64
}

func newRegistry(t *testing.T) *registry {
	t.Helper()
	d := dbtest.Server(t)
	users := authrepo.Repo{DB: d}
	now := time.Now().UTC()
	ann, err := users.InsertUser(ctx, "Ann", "ann@test.com", "h", "user", now)
	if err != nil {
		t.Fatal(err)
	}
	bob, _ := users.InsertUser(ctx, "Bob", "bob@test.com", "h", "user", now)
	r := &registry{db: d, repo: spacesrepo.Repo{DB: d}, ann: ann, bob: bob}
	r.space, err = r.repo.Insert(ctx, "uuid-1", "Home", &ann, now, spacesrepo.Look{})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSpacesAreFoundByIDUUID(t *testing.T) {
	r := newRegistry(t)
	if sp, _ := r.repo.ByID(ctx, r.space); sp == nil || sp.UUID != "uuid-1" || sp.Name != "Home" {
		t.Fatalf("space = %+v", sp)
	}
	if sp, err := r.repo.ByID(ctx, 999); sp != nil || err != nil {
		t.Fatalf("unknown space = %v, %v", sp, err)
	}
	if ok, _ := r.repo.HasUUID(ctx, "uuid-1"); !ok {
		t.Fatal("registered uuid not found")
	}
	if ok, err := r.repo.HasUUID(ctx, "other"); ok || err != nil {
		t.Fatalf("unknown uuid = %v, %v", ok, err)
	}
	if all, _ := r.repo.All(ctx); len(all) != 1 || all[0].UUID != "uuid-1" {
		t.Fatalf("all = %+v", all)
	}
}

func TestMembersRoles(t *testing.T) {
	r := newRegistry(t)
	now := time.Now().UTC()
	if role, err := r.repo.MemberRole(ctx, r.space, r.ann); role != "" || err != nil {
		t.Fatalf("a non-member has role %q, %v", role, err)
	}
	_ = r.repo.UpsertMember(ctx, r.space, r.ann, "admin", now)
	_ = r.repo.UpsertMember(ctx, r.space, r.bob, "viewer", now)
	_ = r.repo.UpsertMember(ctx, r.space, r.bob, "editor", now) // a second upsert changes the role

	if role, _ := r.repo.MemberRole(ctx, r.space, r.bob); role != "editor" {
		t.Fatalf("bob = %q", role)
	}
	if n, _ := r.repo.CountMembers(ctx, r.space); n != 2 {
		t.Fatalf("members = %d", n)
	}
	if n, _ := r.repo.CountWithRole(ctx, r.space, "admin"); n != 1 {
		t.Fatalf("admins = %d", n)
	}
	if ids, _ := r.repo.AdminSpaceIDs(ctx, r.ann); len(ids) != 1 || ids[0] != r.space {
		t.Fatalf("admin spaces = %v", ids)
	}
	if n, _ := r.repo.CountAdminSpaces(ctx, r.bob); n != 0 {
		t.Fatalf("bob administers %d spaces", n)
	}
	list, _ := r.repo.Members(ctx, r.space)
	if len(list) != 2 || list[0].CreatedAt == nil {
		t.Fatalf("members = %+v", list)
	}
	if mine, _ := r.repo.ForUser(ctx, r.bob); len(mine) != 1 || mine[0].Role != "editor" || mine[0].Name != "Home" {
		t.Fatalf("bob's spaces = %+v", mine)
	}

	_ = r.repo.DeleteMember(ctx, r.space, r.bob)
	if role, _ := r.repo.MemberRole(ctx, r.space, r.bob); role != "" {
		t.Fatalf("bob still %q", role)
	}
}

func TestQuotaIsOptional(t *testing.T) {
	r := newRegistry(t)
	if q, err := r.repo.Quota(ctx, r.space); q != nil || err != nil {
		t.Fatalf("default quota = %v, %v", q, err)
	}
	limit := int64(1 << 20)
	_ = r.repo.SetQuota(ctx, r.space, &limit, time.Now())
	if q, _ := r.repo.Quota(ctx, r.space); q == nil || *q != limit {
		t.Fatalf("quota = %v", q)
	}
	_ = r.repo.SetQuota(ctx, r.space, nil, time.Now())
	if q, _ := r.repo.Quota(ctx, r.space); q != nil {
		t.Fatalf("a cleared quota = %v", *q)
	}
}

func TestInvitationsUsedOnceRevokedTheirSpace(t *testing.T) {
	r := newRegistry(t)
	now := time.Now().UTC().Truncate(time.Second)
	if _, err := r.repo.InsertInvitation(ctx, spacesrepo.NewInvitation{
		SpaceID: r.space, InvitedBy: r.ann, Email: "carol@test.com", Role: "viewer", TokenHash: "h1",
		ByServerAdmin: true, Expires: now.Add(time.Hour), Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.repo.InsertInvitation(ctx, spacesrepo.NewInvitation{
		SpaceID: r.space, InvitedBy: r.ann, Role: "editor", TokenHash: "h2", Expires: now.Add(time.Hour), Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	inv, _ := r.repo.InvitationByHash(ctx, "h1")
	if inv == nil || inv.Email == nil || *inv.Email != "carol@test.com" || !inv.ByServerAdmin || inv.Used ||
		inv.SpaceName != "Home" || inv.Inviter != "Ann" || !inv.Expires.Equal(now.Add(time.Hour)) {
		t.Fatalf("invitation = %+v", inv)
	}
	if open, _ := r.repo.InvitationByHash(ctx, "h2"); open == nil || open.Email != nil {
		t.Fatalf("an open invitation has no email: %+v", open)
	}
	if none, err := r.repo.InvitationByHash(ctx, "nope"); none != nil || err != nil {
		t.Fatalf("unknown token = %v, %v", none, err)
	}

	if ok, _ := r.repo.AcceptInvitation(ctx, inv.ID, r.bob, now); !ok {
		t.Fatal("first accept failed")
	}
	if ok, _ := r.repo.AcceptInvitation(ctx, inv.ID, r.bob, now); ok {
		t.Fatal("an invitation must be used only once")
	}
	if used, _ := r.repo.InvitationByHash(ctx, "h1"); !used.Used {
		t.Fatal("the invitation is not marked used")
	}

	list, _ := r.repo.Invitations(ctx, r.space, now)
	if len(list) != 2 || list[0].ID < list[1].ID { // newest first
		t.Fatalf("invitations = %+v", list)
	}
	if ok, _ := r.repo.DeleteAcceptedInvitation(ctx, r.space+1, inv.ID); ok {
		t.Fatal("another space deleted the invitation")
	}
	if ok, _ := r.repo.DeleteAcceptedInvitation(ctx, r.space, list[0].ID); ok {
		t.Fatal("a pending invitation was deleted as if accepted")
	}
	if ok, _ := r.repo.DeleteAcceptedInvitation(ctx, r.space, inv.ID); !ok {
		t.Fatal("the space could not delete its accepted invitation")
	}
}

func TestExpiringInvitationKeepsRowItsCooldown(t *testing.T) {
	r := newRegistry(t)
	now := time.Now().UTC().Truncate(time.Second)
	id, err := r.repo.InsertInvitation(ctx, spacesrepo.NewInvitation{
		SpaceID: r.space, InvitedBy: r.ann, Email: "carol@test.com", Role: "editor", TokenHash: "h1",
		Expires: now.Add(time.Hour), Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	if ok, _ := r.repo.ExpireInvitation(ctx, r.space+1, id, now); ok {
		t.Fatal("another space expired the invitation")
	}
	if ok, _ := r.repo.ExpireInvitation(ctx, r.space, id, now); !ok {
		t.Fatal("the space could not expire its invitation")
	}
	if ok, _ := r.repo.ExpireInvitation(ctx, r.space, id, now); ok {
		t.Fatal("an expired invitation was expired again")
	}
	if list, _ := r.repo.Invitations(ctx, r.space, now); len(list) != 0 {
		t.Fatalf("an expired invitation is still listed: %+v", list)
	}
	held, err := r.repo.UnacceptedFor(ctx, "CAROL@test.com")
	if err != nil || len(held) != 1 || held[0].SpaceID != r.space || held[0].Expires.After(now) {
		t.Fatalf("held = %+v, %v; the expired row must remain, found by address in any case", held, err)
	}
	if n, _ := r.repo.PruneInvitations(ctx, now.Add(-time.Minute)); n != 0 {
		t.Fatalf("pruned %d rows that expired after the cutoff", n)
	}
	if n, _ := r.repo.PruneInvitations(ctx, now.Add(time.Minute)); n != 1 {
		t.Fatalf("pruned %d rows, want 1", n)
	}
}

func TestClearingExpiredInvitationsSparesPendingAccepted(t *testing.T) {
	r := newRegistry(t)
	now := time.Now().UTC().Truncate(time.Second)
	add := func(email, hash string, expires time.Time) int64 {
		id, err := r.repo.InsertInvitation(ctx, spacesrepo.NewInvitation{
			SpaceID: r.space, InvitedBy: r.ann, Email: email, Role: "editor", TokenHash: hash, Expires: expires, Now: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	add("a@test.com", "a", now.Add(-time.Minute))
	add("b@test.com", "b", now.Add(-time.Minute))
	add("c@test.com", "c", now.Add(time.Hour))
	done := add("d@test.com", "d", now.Add(-time.Minute))
	if ok, _ := r.repo.AcceptInvitation(ctx, done, r.bob, now); !ok {
		t.Fatal("accept")
	}

	if n, _ := r.repo.ClearExpiredInvitations(ctx, "A@test.com", now); n != 1 {
		t.Fatalf("cleared %d for one address, want 1", n)
	}
	if n, _ := r.repo.ClearExpiredInvitations(ctx, "", now); n != 1 {
		t.Fatalf("cleared %d for all, want only b", n)
	}
	var left int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM space_invitations`).Scan(&left); err != nil || left != 2 {
		t.Fatalf("left %d, %v; the pending and the accepted row stay", left, err)
	}
}

func TestMemberByEmail(t *testing.T) {
	r := newRegistry(t)
	if err := r.repo.UpsertMember(ctx, r.space, r.ann, "admin", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if ok, err := r.repo.IsMemberByEmail(ctx, r.space, "ann@test.com"); err != nil || !ok {
		t.Fatalf("ann is a member: %v, %v", ok, err)
	}
	if ok, _ := r.repo.IsMemberByEmail(ctx, r.space+1, "ann@test.com"); ok {
		t.Fatal("membership of another space counted")
	}
	if ok, _ := r.repo.IsMemberByEmail(ctx, r.space, "nobody@test.com"); ok {
		t.Fatal("an unknown address is not a member")
	}
}

func TestAuditLogNewestFirstActorOptional(t *testing.T) {
	r := newRegistry(t)
	t0 := time.Now().UTC().Truncate(time.Second)
	_ = r.repo.InsertAudit(ctx, &r.ann, &r.space, &r.bob, "assign_admin", "bob@test.com", t0)
	_ = r.repo.InsertAudit(ctx, nil, &r.space, nil, "set_quota", "", t0.Add(time.Minute))

	log, err := r.repo.Audit(ctx, r.space)
	if err != nil || len(log) != 2 {
		t.Fatalf("log = %+v, %v", log, err)
	}
	if log[0].Action != "set_quota" || log[1].Action != "assign_admin" {
		t.Fatalf("order = %s, %s", log[0].Action, log[1].Action)
	}
	if log[1].Actor != "Ann" || log[1].TargetUserID == nil || *log[1].TargetUserID != r.bob ||
		log[1].Details != "bob@test.com" ||
		log[1].CreatedAt == nil ||
		!log[1].CreatedAt.Equal(t0) {
		t.Fatalf("entry = %+v", log[1])
	}
	if log[0].TargetUserID != nil {
		t.Fatalf("entry without a target = %+v", log[0])
	}
}

func TestOverviewCountsMembersAdmins(t *testing.T) {
	r := newRegistry(t)
	now := time.Now().UTC()
	_ = r.repo.UpsertMember(ctx, r.space, r.ann, "admin", now)
	_ = r.repo.UpsertMember(ctx, r.space, r.bob, "viewer", now)
	rows, err := r.repo.Overview(ctx)
	if err != nil || len(rows) != 1 || rows[0].Members != 2 || rows[0].Admins != 1 || rows[0].CreatedAt == nil {
		t.Fatalf("overview = %+v, %v", rows, err)
	}
}

func strp(s string) *string { return &s }

func TestSpaceLookRoundTrips(t *testing.T) {
	r := newRegistry(t)
	now := time.Now().UTC()
	if sp, _ := r.repo.ByID(ctx, r.space); sp.Emoji != nil || sp.Color != nil {
		t.Fatalf("plain space has a look: %+v", sp)
	}
	id, err := r.repo.Insert(ctx, "uuid-2", "Trip", &r.ann, now, spacesrepo.Look{Emoji: strp("🏖️"), Color: strp("#ff8800")})
	if err != nil {
		t.Fatal(err)
	}
	sp, _ := r.repo.ByID(ctx, id)
	if sp.Emoji == nil || *sp.Emoji != "🏖️" || sp.Color == nil || *sp.Color != "#ff8800" {
		t.Fatalf("inserted look = %+v", sp)
	}

	if err := r.repo.Update(ctx, id, "Trip 2", spacesrepo.Look{Color: strp("#112233")}, now); err != nil {
		t.Fatal(err)
	}
	sp, _ = r.repo.ByID(ctx, id)
	if sp.Name != "Trip 2" || sp.Emoji != nil || *sp.Color != "#112233" {
		t.Fatalf("updated = %+v", sp)
	}

	_ = r.repo.UpsertMember(ctx, id, r.ann, "admin", now)
	mine, _ := r.repo.ForUser(ctx, r.ann)
	var found bool
	for _, s := range mine {
		if s.ID == id {
			found = *s.Color == "#112233"
		}
	}
	if !found {
		t.Fatalf("ForUser lost the look: %+v", mine)
	}
	ov, _ := r.repo.Overview(ctx)
	for _, o := range ov {
		if o.ID == id && *o.Color != "#112233" {
			t.Fatalf("Overview lost the look: %+v", o)
		}
	}
	all, _ := r.repo.All(ctx)
	for _, s := range all {
		if s.ID == id && *s.Color != "#112233" {
			t.Fatalf("All lost the look: %+v", s)
		}
	}
}

func TestSpaceColorIsCheckedByTheSchema(t *testing.T) {
	r := newRegistry(t)
	_, err := r.repo.Insert(ctx, "uuid-3", "Bad", &r.ann, time.Now().UTC(), spacesrepo.Look{Color: strp("red")})
	if err == nil {
		t.Fatal("a color that is not #rrggbb was stored")
	}
}
