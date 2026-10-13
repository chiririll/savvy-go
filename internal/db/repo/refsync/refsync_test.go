package refsync_test

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	authrepo "savvy-go/internal/db/repo/auth"
	repo "savvy-go/internal/db/repo/refsync"
)

var ctx = context.Background()

func TestGroupsOnTheServer(t *testing.T) {
	d := dbtest.Server(t)
	now := time.Now().UTC()
	ann, _ := authrepo.Repo{DB: d}.InsertUser(ctx, "Ann", "ann@test.com", "h", "user", now)
	for _, uuid := range []string{"a", "b", "c"} {
		if _, err := d.ExecContext(ctx, `INSERT INTO spaces (uuid, name, created_by) VALUES (?, ?, ?)`, uuid, uuid, ann); err != nil {
			t.Fatal(err)
		}
	}
	s := repo.Server{DB: d}

	if g, err := s.GroupOf(ctx, 1); err != nil || g != nil {
		t.Fatalf("group of a lone space = %+v, %v", g, err)
	}
	id, err := s.InsertGroup(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, sp := range []int64{1, 2} {
		if err := s.AddSpace(ctx, id, sp); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.AddSpace(ctx, id, 1); err == nil {
		t.Fatal("a space joined the same group twice")
	}
	other, _ := s.InsertGroup(ctx, now)
	if err := s.AddSpace(ctx, other, 2); err == nil {
		t.Fatal("a space is in two groups")
	}

	g, err := s.GroupOf(ctx, 2)
	if err != nil || g == nil || g.ID != id || g.Categories || g.Tags {
		t.Fatalf("group = %+v, %v", g, err)
	}
	if err := s.SetFlags(ctx, id, true, false); err != nil {
		t.Fatal(err)
	}
	if g, _ = s.GroupOf(ctx, 1); !g.Categories || g.Tags {
		t.Fatalf("flags = %+v", g)
	}
	if g, _ = s.Group(ctx, id); g == nil || g.ID != id || !g.Categories {
		t.Fatalf("group by id = %+v", g)
	}
	if g, _ = s.Group(ctx, 999); g != nil {
		t.Fatalf("group 999 = %+v", g)
	}
	if all, _ := s.Groups(ctx); len(all) != 2 || all[0].ID != id || !all[0].Categories {
		t.Fatalf("groups = %+v", all)
	}
	members, _ := s.Members(ctx, id)
	if len(members) != 2 || members[0] != 1 || members[1] != 2 {
		t.Fatalf("members = %v", members)
	}

	if ok, _ := s.RemoveSpace(ctx, id, 1); !ok {
		t.Fatal("remove failed")
	}
	if ok, _ := s.RemoveSpace(ctx, id, 1); ok {
		t.Fatal("removing twice must report that nothing was removed")
	}
	if err := s.DeleteGroup(ctx, id); err != nil {
		t.Fatal(err)
	}
	if g, _ := s.GroupOf(ctx, 2); g != nil {
		t.Fatalf("the group outlived its deletion: %+v", g)
	}

	// Deleting a space takes it out of its group.
	if err := s.AddSpace(ctx, other, 3); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ExecContext(ctx, `DELETE FROM spaces WHERE id = 3`); err != nil {
		t.Fatal(err)
	}
	if members, _ := s.Members(ctx, other); len(members) != 0 {
		t.Fatalf("members after the space was deleted = %v", members)
	}
}
