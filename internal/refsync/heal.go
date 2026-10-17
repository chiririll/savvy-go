package refsync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"savvy-go/internal/store"
)

// Heal brings the spaces of every sync group back together after they drifted
// apart: a crash between the commits of two spaces, a restored backup, a space
// that was unavailable while the others changed. Nobody is there to decide, so
// it only ever adds: whatever a space has and another lacks is created there,
// never deleted, and where spaces disagree the lowest space id wins (its icon
// and color, its default category). Deliberate deletions are not lost by this,
// because a delete reaches every space in one transaction.
//
// A group that cannot be healed (a space is unavailable) is skipped and
// reported in the error; the others are still done.
func (s Service) Heal(ctx context.Context) error {
	groups, err := s.server().Groups(ctx)
	if err != nil {
		return err
	}
	var failed []error
	for _, g := range groups {
		members, err := s.server().Members(ctx, g.ID)
		if err != nil {
			return err
		}
		var kinds []Kind
		if g.Categories {
			kinds = append(kinds, Categories)
		}
		if g.Tags {
			kinds = append(kinds, Tags)
		}
		if len(members) < 2 || len(kinds) == 0 {
			continue
		}
		err = s.st().InSpaces(ctx, members, func(dbs map[int64]store.DB) error {
			diff, err := diffSpaces(ctx, dbs, members, kinds)
			if err != nil || diff.Empty() {
				return err
			}
			slog.Info("sync group had drifted apart; adding what is missing", "group", g.ID,
				"categories", len(diff.Categories), "tags", len(diff.Tags), "defaults", len(diff.Defaults))
			return reconcileSpaces(ctx, dbs, members, kinds, keepEverything(diff))
		})
		if err != nil {
			failed = append(failed, fmt.Errorf("sync group %d: %w", g.ID, err))
		}
	}
	return errors.Join(failed...)
}

// keepEverything decides every difference in the way that loses nothing.
// Present lists the spaces by ascending id, so its first is the lowest.
func keepEverything(d Diff) Plan {
	var p Plan
	for _, c := range d.Categories {
		p.Decisions = append(
			p.Decisions,
			Decision{Kind: Categories, Type: c.Type, Name: c.Name, Keep: true, From: c.Present[0].Space},
		)
	}
	for _, t := range d.Tags {
		p.Decisions = append(p.Decisions, Decision{Kind: Tags, Name: t.Name, Keep: true})
	}
	for _, c := range d.Defaults {
		for _, id := range slices.Sorted(maps.Keys(c.Defaults)) {
			if name := c.Defaults[id]; name != "" {
				p.Defaults = append(p.Defaults, DefaultDecision{Type: c.Type, Name: name})
				break
			}
		}
	}
	return p
}
