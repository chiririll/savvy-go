package seed

import (
	"fmt"

	"savvy-go/internal/refsync"
)

// syncedSpaces are the spaces of the demo sync group. Demo stays out, so
// adding it has categories and tags to reconcile.
var syncedSpaces = []string{spaceFamily, spaceStudio}

// syncReference puts syncedSpaces in one sync group that keeps categories and
// tags equal. Each of them has a category and a tag of its own, which the group
// keeps in both.
func (d *demo) syncReference() error {
	svc := refsync.Service{Spaces: d.spaces}
	alex := d.people[userAlex]
	ids := map[int64]bool{}
	var members []int64
	for _, key := range syncedSpaces {
		id := d.space(key).ID
		ids[id] = true
		members = append(members, id)
	}
	change := refsync.Change{Add: members, Categories: true, Tags: true}
	diff, err := svc.Diff(d.ctx, alex, change)
	if err != nil {
		return fmt.Errorf("sync group diff: %w", err)
	}
	plan := keepAll(diff)
	plan.Change = change
	g, err := svc.Apply(d.ctx, alex, plan)
	if err != nil {
		return fmt.Errorf("sync group: %w", err)
	}
	for i := range d.manifest.Spaces {
		if sp := &d.manifest.Spaces[i]; ids[sp.ID] {
			sp.SyncGroup = g.ID
		}
	}
	return nil
}

// keepAll decides every difference by keeping the item in all spaces. The
// presets are the same everywhere, so only the spaces' own items differ.
func keepAll(d refsync.Diff) refsync.Plan {
	var p refsync.Plan
	for _, c := range d.Categories {
		p.Decisions = append(p.Decisions, refsync.Decision{
			Kind: refsync.Categories, Type: c.Type, Name: c.Name, Keep: true, From: c.Present[0].Space,
		})
	}
	for _, t := range d.Tags {
		p.Decisions = append(p.Decisions, refsync.Decision{Kind: refsync.Tags, Name: t.Name, Keep: true})
	}
	return p
}
