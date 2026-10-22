package settings_test

import (
	"context"
	"testing"

	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/settings"
)

func TestServerSettingsFallBackTheirDefaults(t *testing.T) {
	ctx := context.Background()
	s := settings.Store{DB: dbtest.Server(t)}

	if s.Get(ctx, "password_login_enabled", nil) != true {
		t.Error("password login is on until an admin turns it off")
	}
	if got := s.Get(ctx, "space_backups_max", nil); got != 10 {
		t.Errorf("default backups kept: %v", got)
	}
	if got := s.Get(ctx, "no_such_key", nil); got != nil {
		t.Errorf("an unknown key is nil: %v", got)
	}
	if got := s.Get(ctx, "no_such_key", "x"); got != "x" {
		t.Errorf("the caller's fallback is used for an unknown key: %v", got)
	}
}

func TestStoredValuesWinKeepTheirType(t *testing.T) {
	ctx := context.Background()
	s := settings.Store{DB: dbtest.Server(t)}
	for key, value := range map[string]any{"space_quota_mb": 25.5, "sso_allow_signup": false, "max_spaces_per_user": nil} {
		if err := s.Set(ctx, key, value); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.Get(ctx, "space_quota_mb", nil); got != 25.5 {
		t.Errorf("number: %v", got)
	}
	if s.Bool(ctx, "sso_allow_signup", true) {
		t.Error("a stored false must beat a true fallback")
	}
	if got := s.Get(ctx, "max_spaces_per_user", 5); got != nil {
		t.Errorf("a stored null means unlimited, not the fallback: %v", got)
	}
	if err := s.Set(ctx, "bad", make(chan int)); err == nil {
		t.Error("a value that is not JSON was stored")
	}
}

func TestBoolReadsFormsSettingCanTake(t *testing.T) {
	ctx := context.Background()
	s := settings.Store{DB: dbtest.Server(t)}
	for _, tc := range []struct {
		stored any
		want   bool
	}{{true, true}, {false, false}, {float64(1), true}, {float64(0), false}, {"true", true}, {"false", false},
		{"T", true}} {
		if err := s.Set(ctx, "flag", tc.stored); err != nil {
			t.Fatal(err)
		}
		if got := s.Bool(ctx, "flag", true); got != tc.want {
			t.Errorf("Bool of %#v = %v, want %v", tc.stored, got, tc.want)
		}
	}
	if err := s.Set(ctx, "flag", "banana"); err != nil {
		t.Fatal(err)
	}
	if s.Bool(ctx, "flag", false) {
		t.Error("an unreadable value gives the fallback")
	}
	if s.Bool(ctx, "never_set", false) || !s.Bool(ctx, "never_set", true) {
		t.Error("an unset key gives the fallback")
	}
}

func TestInt(t *testing.T) {
	ctx := context.Background()
	s := settings.Store{DB: dbtest.Server(t)}
	if got := s.Int(ctx, "space_backups_max", 3); got != 10 {
		t.Errorf("an unset known key reads as its default: %d", got)
	}
	if got := s.Int(ctx, "no_such_key", 3); got != 3 {
		t.Errorf("an unknown key reads as the fallback: %d", got)
	}
	for _, tc := range []struct {
		stored any
		want   int64
	}{{float64(7), 7}, {"12", 12}, {"x", 99}, {true, 99}} {
		if err := s.Set(ctx, "n", tc.stored); err != nil {
			t.Fatal(err)
		}
		if got := s.Int(ctx, "n", 99); got != tc.want {
			t.Errorf("Int of %#v = %d, want %d", tc.stored, got, tc.want)
		}
	}
}

func TestAllListsOnlyKnownKeysKind(t *testing.T) {
	ctx := context.Background()
	server, space := settings.Store{DB: dbtest.Server(t)}, settings.Store{DB: dbtest.Space(t), Space: true}
	for _, kv := range [][2]any{{"space_quota_mb", 5}, {"legacy_marker", 1}, {"stray", 1},
		{"auto_update_currencies", false}} {
		if err := server.Set(ctx, kv[0].(string), kv[1]); err != nil {
			t.Fatal(err)
		}
	}
	all, err := server.All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if all["space_quota_mb"] != float64(5) || all["password_login_enabled"] != true {
		t.Errorf("stored and default values: %v", all)
	}
	for _, hidden := range []string{"legacy_marker", "stray", "auto_update_currencies"} {
		if _, ok := all[hidden]; ok {
			t.Errorf("%s is not a server setting and must stay private", hidden)
		}
	}

	if err := space.Set(ctx, "auto_update_currencies", false); err != nil {
		t.Fatal(err)
	}
	spaceAll, _ := space.All(ctx)
	if spaceAll["auto_update_currencies"] != false || len(spaceAll) != 1 {
		t.Errorf("space settings %v", spaceAll)
	}
	if !space.Bool(ctx, "never_stored", true) {
		t.Error("space fallback")
	}
}

func TestKeyKinds(t *testing.T) {
	if !settings.IsServerKey("space_quota_mb") || settings.IsServerKey("auto_update_currencies") ||
		!settings.IsSpaceKey("auto_update_currencies") ||
		settings.IsSpaceKey("space_quota_mb") ||
		settings.IsServerKey("nope") {
		t.Error("a key belongs to the server or to a space, never both")
	}
}

func TestSpaceQuotaInBytes(t *testing.T) {
	ctx := context.Background()
	d := dbtest.Server(t)
	s := settings.Store{DB: d}
	if got := settings.SpaceQuota(ctx, d); got != 0 {
		t.Errorf("unlimited by default: %d", got)
	}
	for _, tc := range []struct {
		stored any
		want   int64
	}{{float64(2), 2 << 20}, {0.5, 1 << 19}, {"3", 3 << 20}, {float64(-4), 0}, {"junk", 0}, {nil, 0}} {
		if err := s.Set(ctx, "space_quota_mb", tc.stored); err != nil {
			t.Fatal(err)
		}
		if got := settings.SpaceQuota(ctx, d); got != tc.want {
			t.Errorf("quota for %#v = %d, want %d", tc.stored, got, tc.want)
		}
	}
}
