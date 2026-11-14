package seed

import (
	"context"
	"strings"
	"testing"
	"time"

	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/signing"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite/sqlitetest"
)

func demoWith(t *testing.T, opts Options) (*Manifest, store.Store) {
	t.Helper()
	key, err := signing.Generate(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	opts.Now = time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	st := sqlitetest.Open(t)
	m, err := Demo(context.Background(), st, signing.KeyRing{Holder: signing.NewHolder(key)}, opts)
	if err != nil || m == nil {
		t.Fatalf("Demo = %v, %v", m, err)
	}
	return m, st
}

func TestDemoCopiesAddUsersAndSpaces(t *testing.T) {
	m, _ := demoWith(t, Options{Months: 1, Copies: 2})
	if len(m.Users) != 9 || len(m.Spaces) != 9 {
		t.Fatalf("%d users, %d spaces, want 9 and 9", len(m.Users), len(m.Spaces))
	}
	emails := map[string]bool{}
	for _, u := range m.Users {
		emails[u.Email] = true
	}
	for _, e := range []string{"admin@savvy.app", "admin1@savvy.app", "admin2@savvy.app"} {
		if !emails[e] {
			t.Errorf("no user %s", e)
		}
	}
	names := map[string]bool{}
	for _, sp := range m.Spaces {
		names[sp.Name] = true
	}
	if !names["Demo"] || !names["Demo 1"] || !names["Studio GmbH 2"] {
		t.Errorf("space names: %v", names)
	}
	// Ids are per space, so every Demo space lists the same rules, no more.
	var want int
	for _, sp := range m.Spaces {
		if sp.Name == "Demo" {
			want = len(sp.Automations)
		}
	}
	for _, sp := range m.Spaces {
		if len(sp.Automations) != 0 && strings.HasPrefix(sp.Name, "Demo") && len(sp.Automations) != want {
			t.Errorf("%s lists %d automations, want %d", sp.Name, len(sp.Automations), want)
		}
	}
}

func TestDemoMonthsSetsHistory(t *testing.T) {
	short := countDemoTx(t, Options{Months: 2})
	long := countDemoTx(t, Options{Months: 24})
	if long < short*5 {
		t.Fatalf("24 months gave %d transactions, 2 months %d", long, short)
	}
}

func countDemoTx(t *testing.T, opts Options) int {
	t.Helper()
	m, st := demoWith(t, opts)
	for _, sp := range m.Spaces {
		if sp.Name == "Demo" {
			db, err := st.Space(context.Background(), sp.ID)
			if err != nil {
				t.Fatal(err)
			}
			return ledgerrepo.Repo{DB: db}.CountTransactions(context.Background())
		}
	}
	t.Fatal("no Demo space")
	return 0
}

func TestLoadConfigRejectsNegative(t *testing.T) {
	for _, body := range []string{"months = -1", "copies = -2"} {
		if _, err := LoadConfig(writeSeedConfig(t, body)); err == nil {
			t.Errorf("%q accepted", body)
		}
	}
	c, err := LoadConfig(writeSeedConfig(t, "months = 120\ncopies = 3"))
	if err != nil || c.Months != 120 || c.Copies != 3 {
		t.Fatalf("got %+v, %v", c, err)
	}
}
