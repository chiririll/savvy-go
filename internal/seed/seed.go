package seed

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/settings"
	"savvy-go/internal/signing"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/transfers"
)

const demoSeededKey = "demo_seeded"

// Options configures Demo.
type Options struct {
	// Loc is the time zone of the seeded dates; UTC when nil.
	Loc *time.Location
	// Now is the moment the data is placed relative to; the current time when
	// zero. Pin it to seed the same data on any day.
	Now time.Time
	// Months is how many months of history the Demo space gets; 12 when zero.
	Months int
	// Copies is how many extra sets of users and spaces to create next to the
	// regular one: admin1, editor1, ... for the first, admin2, ... for the
	// second. Handy to load the server for performance tests.
	Copies int
}

// Demo seeds demo users and several spaces when the server has
// never been demo-seeded (no users yet). Later starts are no-ops so first-boot
// matches Laravel's SEED_DEMO behavior.
//
// The spaces (demoSpaces) cover every feature of the spaces model:
//   - "Demo": the full workspace in USD with a EUR account; Alex administers it,
//     Jordan edits it and the demo user only views it.
//   - "Family Budget": Alex and Jordan administer it, the guest Sam views it.
//     It is linked to Demo.
//   - "Studio GmbH": a EUR-based space Alex administers, Jordan only views. It is
//     linked to Demo, so transfers between them cross currencies, and it is not
//     linked to Family Budget. It has open invitations.
//
// Every space has an emoji and a color. Family Budget and Demo each have a
// category and a tag of their own and share them in a sync group, with Studio
// GmbH outside it: Studio has a business set of categories and tags instead of
// the presets. Sam's address is unverified.
//
// It returns what it created, or nil when it seeded nothing.
func Demo(ctx context.Context, st store.Store, keys signing.KeyRing, opts Options) (*Manifest, error) {
	loc := opts.Loc
	if loc == nil {
		loc = time.UTC
	}
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	now = now.In(loc)
	server := settings.Store{DB: st.Server()}
	users := auth.Users{DB: st.Server()}
	if skip, err := alreadySeeded(ctx, server, users); skip || err != nil {
		return nil, err
	}

	months := opts.Months
	if months <= 0 {
		months = monthsOfHistory
	}
	if opts.Copies < 0 {
		return nil, fmt.Errorf("seed copies %d: want 0 or more", opts.Copies)
	}
	manifest := &Manifest{Now: now}
	started := time.Now()
	slog.Info("demo seed started", "date", now.Format("2006-01-02"), "months", months, "copies", opts.Copies)
	for n := 0; n <= opts.Copies; n++ {
		err := timed("demo set", func() error {
			return seedSet(ctx, st, keys, users, manifest, n, now, months)
		}, "set", n, "of", opts.Copies)
		if err != nil {
			return nil, err
		}
	}
	if err := server.Set(ctx, demoSeededKey, true); err != nil {
		return nil, fmt.Errorf("mark demo seeded: %w", err)
	}
	slog.Info("demo seed done", "users", len(manifest.Users), "spaces", len(manifest.Spaces),
		"took", time.Since(started).Round(time.Millisecond))
	return manifest, nil
}

// seedSet creates copy n of the users and spaces (0 is the regular set) and
// adds what it made to manifest.
func seedSet(
	ctx context.Context, st store.Store, keys signing.KeyRing, users auth.Users,
	manifest *Manifest, n int, now time.Time, months int,
) error {
	people, made, err := createDemoUsers(ctx, users, n)
	if err != nil {
		return err
	}
	slog.Info("seed users created", "set", n, "count", len(made), "first", made[0].email)
	d := &demo{
		ctx: ctx, st: st, spaces: spaces.Service{Store: st}, people: people, now: now, months: months,
		manifest: manifest, nameSuffix: copySuffix(n),
	}
	for _, u := range made {
		manifest.Users = append(manifest.Users, ManifestUser{
			Key: u.key, Name: u.name, Email: u.email, Password: u.pass, Role: u.role, Verified: !u.unverified,
		})
	}
	steps := []struct {
		name string
		run  func() error
	}{
		{"create spaces", d.createSpaces},
		{"sync reference data", d.syncReference},
		{"link spaces and transfer", func() error { return d.linkTransfer(transfers.Service{Spaces: d.spaces, Keys: keys}) }},
		{"reconcile", d.reconcile},
		{"invite", d.invite},
		{"record automations", d.recordAutomations},
	}
	for _, step := range steps {
		if err := timed("seed "+step.name, step.run, "set", n); err != nil {
			return err
		}
	}
	return d.logSeeded()
}

// copySuffix is what is added to the names of copy n: nothing for the regular set.
func copySuffix(n int) string {
	if n == 0 {
		return ""
	}
	return " " + strconv.Itoa(n)
}

// alreadySeeded reports whether Demo has nothing to do: it ran before, or the
// server already has users.
func alreadySeeded(ctx context.Context, server settings.Store, users auth.Users) (bool, error) {
	if server.Bool(ctx, demoSeededKey, false) {
		return true, nil
	}
	n, err := users.Count(ctx)
	if err != nil {
		return false, fmt.Errorf("count users: %w", err)
	}
	return n > 0, nil
}
