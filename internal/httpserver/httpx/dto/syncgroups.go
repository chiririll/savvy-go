package dto

import (
	"slices"

	"savvy-go/internal/refsync"
)

// SpaceName is a space by its name only.
type SpaceName struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// SyncGroup is a group of spaces that keep categories and tags equal.
type SyncGroup struct {
	ID         int64   `json:"id"`
	Categories bool    `json:"categories"`
	Tags       bool    `json:"tags"`
	Members    []int64 `json:"members"`
}

func NewSyncGroup(g refsync.Group) SyncGroup {
	return SyncGroup{ID: g.ID, Categories: g.Categories, Tags: g.Tags, Members: orEmpty(g.Members)}
}

// SyncGroupView is a sync group as one user sees it: the spaces they belong to
// by name and role, and a count of those they do not.
type SyncGroupView struct {
	ID         int64   `json:"id"`
	Categories bool    `json:"categories"`
	Tags       bool    `json:"tags"`
	Members    []Space `json:"members"`
	Hidden     int     `json:"hidden"`
}

type SyncCategoryPresence struct {
	SpaceID      int64   `json:"spaceId"`
	Icon         *string `json:"icon"`
	Color        *string `json:"color"`
	IsDefault    bool    `json:"isDefault"`
	Transactions int     `json:"transactions"`
}

type SyncCategoryConflict struct {
	Type    string                 `json:"type"`
	Name    string                 `json:"name"`
	Present []SyncCategoryPresence `json:"present"`
	Missing []int64                `json:"missing"`
}

type SyncTagPresence struct {
	SpaceID      int64 `json:"spaceId"`
	Transactions int   `json:"transactions"`
}

type SyncTagConflict struct {
	Name    string            `json:"name"`
	Present []SyncTagPresence `json:"present"`
	Missing []int64           `json:"missing"`
}

type SyncDefault struct {
	SpaceID int64  `json:"spaceId"`
	Name    string `json:"name"`
}

type SyncDefaultConflict struct {
	Type     string        `json:"type"`
	Defaults []SyncDefault `json:"defaults"`
}

// SyncDiff is what makes the spaces of a group differ, with the names of the
// spaces it mentions.
type SyncDiff struct {
	Categories []SyncCategoryConflict `json:"categories"`
	Tags       []SyncTagConflict      `json:"tags"`
	Defaults   []SyncDefaultConflict  `json:"defaults"`
	Spaces     []SpaceName            `json:"spaces"`
}

// NewSyncDiff converts d. name finds a space's name; a space it does not know
// is left out of Spaces.
func NewSyncDiff(d refsync.Diff, name func(id int64) (string, bool)) SyncDiff {
	names := map[int64]string{}
	note := func(ids ...int64) {
		for _, id := range ids {
			if _, seen := names[id]; seen {
				continue
			}
			if n, ok := name(id); ok {
				names[id] = n
			}
		}
	}
	out := SyncDiff{
		Categories: []SyncCategoryConflict{}, Tags: []SyncTagConflict{},
		Defaults: []SyncDefaultConflict{}, Spaces: []SpaceName{},
	}
	for _, c := range d.Categories {
		present := []SyncCategoryPresence{}
		for _, p := range c.Present {
			note(p.Space)
			present = append(present, SyncCategoryPresence{
				SpaceID: p.Space, Icon: p.Icon, Color: p.Color, IsDefault: p.IsDefault, Transactions: p.Transactions,
			})
		}
		note(c.Missing...)
		out.Categories = append(
			out.Categories,
			SyncCategoryConflict{Type: c.Type, Name: c.Name, Present: present, Missing: orEmpty(c.Missing)},
		)
	}
	for _, t := range d.Tags {
		present := []SyncTagPresence{}
		for _, p := range t.Present {
			note(p.Space)
			present = append(present, SyncTagPresence{SpaceID: p.Space, Transactions: p.Transactions})
		}
		note(t.Missing...)
		out.Tags = append(out.Tags, SyncTagConflict{Name: t.Name, Present: present, Missing: orEmpty(t.Missing)})
	}
	for _, c := range d.Defaults {
		list := []SyncDefault{}
		for _, id := range sortedKeys(c.Defaults) {
			note(id)
			list = append(list, SyncDefault{SpaceID: id, Name: c.Defaults[id]})
		}
		out.Defaults = append(out.Defaults, SyncDefaultConflict{Type: c.Type, Defaults: list})
	}
	for _, id := range sortedKeys(names) {
		out.Spaces = append(out.Spaces, SpaceName{ID: id, Name: names[id]})
	}
	return out
}

func sortedKeys[V any](m map[int64]V) []int64 {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
