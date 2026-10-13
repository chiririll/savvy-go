package spaces_test

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	authrepo "savvy-go/internal/db/repo/auth"
	spacesrepo "savvy-go/internal/db/repo/spaces"
)

var ctx = context.Background()

// registry is a server database with two users and one space owned by the first.
type registry struct {
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
	r := &registry{repo: spacesrepo.Repo{DB: d}, ann: ann, bob: bob}
	r.space, err = r.repo.Insert(ctx, "uuid-1", "Home", &ann, now)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSpacesAreFoundByIDAndUUID(t *testing.T) {
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

func TestMembersAndRoles(t *testing.T) {
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

func TestInvitationsAreUsedOnceAndRevokedByTheirSpace(t *testing.T) {
	r := newRegistry(t)
	now := time.Now().UTC().Truncate(time.Second)
	if err := r.repo.InsertInvitation(ctx, spacesrepo.NewInvitation{
		SpaceID: r.space, InvitedBy: r.ann, Email: "carol@test.com", Role: "viewer", TokenHash: "h1",
		ByServerAdmin: true, Expires: now.Add(time.Hour), Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.repo.InsertInvitation(ctx, spacesrepo.NewInvitation{
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

	list, _ := r.repo.Invitations(ctx, r.space)
	if len(list) != 2 || list[0].ID < list[1].ID { // newest first
		t.Fatalf("invitations = %+v", list)
	}
	if ok, _ := r.repo.DeleteInvitation(ctx, r.space+1, inv.ID); ok {
		t.Fatal("another space deleted the invitation")
	}
	if ok, _ := r.repo.DeleteInvitation(ctx, r.space, inv.ID); !ok {
		t.Fatal("the space could not delete its invitation")
	}
}

func TestAuditLogIsNewestFirstAndActorIsOptional(t *testing.T) {
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
	if log[1].Actor != "Ann" || log[1].TargetUserID == nil || *log[1].TargetUserID != r.bob || log[1].Details != "bob@test.com" ||
		log[1].CreatedAt == nil || !log[1].CreatedAt.Equal(t0) {
		t.Fatalf("entry = %+v", log[1])
	}
	if log[0].TargetUserID != nil {
		t.Fatalf("entry without a target = %+v", log[0])
	}
}

func TestOverviewCountsMembersAndAdmins(t *testing.T) {
	r := newRegistry(t)
	now := time.Now().UTC()
	_ = r.repo.UpsertMember(ctx, r.space, r.ann, "admin", now)
	_ = r.repo.UpsertMember(ctx, r.space, r.bob, "viewer", now)
	rows, err := r.repo.Overview(ctx)
	if err != nil || len(rows) != 1 || rows[0].Members != 2 || rows[0].Admins != 1 || rows[0].CreatedAt == nil {
		t.Fatalf("overview = %+v, %v", rows, err)
	}
}
