package settings_test

import (
	"context"
	"testing"

	"savvy-go/internal/db/dbtest"
	settingsrepo "savvy-go/internal/db/repo/settings"
)

func TestSettingsRoundTripOnServerSpace(t *testing.T) {
	ctx := context.Background()
	for name, repo := range map[string]settingsrepo.Repo{
		"server": {DB: dbtest.Server(t)},
		"space":  {DB: dbtest.Space(t), Space: true},
	} {
		if v, ok, err := repo.Get(ctx, "k"); v != "" || ok || err != nil {
			t.Fatalf("%s: unset key = %q, %v, %v", name, v, ok, err)
		}
		if err := repo.Set(ctx, "k", `{"a":1}`); err != nil {
			t.Fatal(err)
		}
		if err := repo.Set(ctx, "k", `true`); err != nil { // a second Set replaces
			t.Fatal(err)
		}
		if v, ok, _ := repo.Get(ctx, "k"); v != "true" || !ok {
			t.Fatalf("%s: k = %q, %v", name, v, ok)
		}
		all, err := repo.All(ctx)
		if err != nil || all["k"] != "true" {
			t.Fatalf("%s: all = %v, %v", name, all, err)
		}
	}
}

func TestSpaceSettingsCanBeDeleted(t *testing.T) {
	ctx := context.Background()
	repo := settingsrepo.Repo{DB: dbtest.Space(t), Space: true}
	_ = repo.Set(ctx, "mark", "1")
	if err := repo.Delete(ctx, "mark"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := repo.Get(ctx, "mark"); ok {
		t.Fatal("the setting is still there")
	}
	if err := repo.Delete(ctx, "mark"); err != nil {
		t.Fatalf("deleting a key that is not set must not fail: %v", err)
	}
}
