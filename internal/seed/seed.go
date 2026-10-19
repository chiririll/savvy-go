package seed

import (
	"context"
	"fmt"
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

	people, err := createDemoUsers(ctx, users)
	if err != nil {
		return nil, err
	}
	d := &demo{
		ctx: ctx, st: st, spaces: spaces.Service{Store: st}, people: people, now: now,
		manifest: &Manifest{Now: now},
	}
	for _, u := range demoUsers {
		d.manifest.Users = append(
			d.manifest.Users,
			ManifestUser{
				Key: u.key, Name: u.name, Email: u.email, Password: u.pass, Role: u.role,
				Verified: !u.unverified,
			},
		)
	}

	steps := []func() error{
		d.createSpaces,
		d.syncReference,
		func() error { return d.linkTransfer(transfers.Service{Spaces: d.spaces, Keys: keys}) },
		d.invite,
		d.recordAutomations,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return nil, err
		}
	}
	if err := server.Set(ctx, demoSeededKey, true); err != nil {
		return nil, fmt.Errorf("mark demo seeded: %w", err)
	}
	if err := d.logSeeded(); err != nil {
		return nil, err
	}
	return d.manifest, nil
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
