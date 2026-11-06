package transfers_test

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	authrepo "savvy-go/internal/db/repo/auth"
	transfersrepo "savvy-go/internal/db/repo/transfers"
)

var ctx = context.Background()

func TestLinksTrustedKeysOnTheServer(t *testing.T) {
	d := dbtest.Server(t)
	users := authrepo.Repo{DB: d}
	now := time.Now().UTC()
	ann, _ := users.InsertUser(ctx, "Ann", "ann@test.com", "h", "user", now)
	// Two registered spaces to link.
	for _, uuid := range []string{"a", "b"} {
		if _,
			err := d.ExecContext(ctx, `INSERT INTO spaces (uuid, name, created_by) VALUES (?, ?, ?)`, uuid, uuid,
			ann); err != nil {
			t.Fatal(err)
		}
	}
	s := transfersrepo.Server{DB: d}

	if ok, _ := s.Linked(ctx, 1, 2); ok {
		t.Fatal("linked before linking")
	}
	if err := s.InsertLink(ctx, 1, 2, ann, now); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.Linked(ctx, 1, 2); !ok {
		t.Fatal("not linked after linking")
	}
	if links, _ := s.Links(ctx, 2); len(links) != 1 || links[0].A != 1 || links[0].B != 2 {
		t.Fatalf("links = %+v", links)
	}
	if all, _ := s.AllLinks(ctx); len(all) != 1 {
		t.Fatalf("all links = %+v", all)
	}
	if ok, _ := s.DeleteLink(ctx, 1, 2); !ok {
		t.Fatal("unlink failed")
	}
	if ok, _ := s.DeleteLink(ctx, 1, 2); ok {
		t.Fatal("unlinking twice must report that nothing was linked")
	}

	if _, ok := s.PublicKey(ctx, "kid"); ok {
		t.Fatal("an unknown key is trusted")
	}
	if err := s.InsertTrustedKey(ctx, "kid", "cHVi", "Other server", ann, now); err != nil {
		t.Fatal(err)
	}
	if key, ok := s.PublicKey(ctx, "kid"); !ok || key != "cHVi" {
		t.Fatalf("key = %q, %v", key, ok)
	}
	if keys, _ := s.TrustedKeys(ctx); len(keys) != 1 || keys[0].Name != "Other server" || keys[0].CreatedAt == nil {
		t.Fatalf("keys = %+v", keys)
	}
	if ok, _ := s.DeleteTrustedKey(ctx, "kid"); !ok {
		t.Fatal("untrust failed")
	}
	if ok, _ := s.DeleteTrustedKey(ctx, "kid"); ok {
		t.Fatal("untrusting twice must report that nothing was trusted")
	}
}
