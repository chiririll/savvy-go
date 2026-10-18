package refsync

import (
	"context"
	"slices"
	"strings"

	"savvy-go/internal/ledger"
	"savvy-go/internal/store"
)

func lower(s string) string { return strings.ToLower(s) }

func sameText(a, b *string) bool {
	x, y := "", ""
	if a != nil {
		x = *a
	}
	if b != nil {
		y = *b
	}
	return x == y
}

func diffSpaces(ctx context.Context, dbs map[int64]store.DB, members []int64, kinds []Kind) (Diff, error) {
	var out Diff
	for _, k := range kinds {
		switch k {
		case Categories:
			conflicts, defaults, err := diffCategories(ctx, dbs, members)
			if err != nil {
				return Diff{}, err
			}
			out.Categories, out.Defaults = conflicts, defaults
		case Tags:
			conflicts, err := diffTags(ctx, dbs, members)
			if err != nil {
				return Diff{}, err
			}
			out.Tags = conflicts
		}
	}
	return out, nil
}

// categoryKey identifies a category across spaces: its type and case-folded name.
type categoryKey struct{ typ, name string }

// categoryView is every space's categories grouped by key.
type categoryView struct {
	states   map[categoryKey][]CategoryState
	names    map[categoryKey]string
	defaults map[string]map[int64]string // type -> space -> default name
}

func gatherCategories(ctx context.Context, dbs map[int64]store.DB, members []int64) (categoryView, error) {
	v := categoryView{
		states:   map[categoryKey][]CategoryState{},
		names:    map[categoryKey]string{},
		defaults: map[string]map[int64]string{},
	}
	for _, id := range members {
		list, err := (ledger.Categories{DB: dbs[id]}).All(ctx, "")
		if err != nil {
			return v, err
		}
		for _, c := range list {
			k := categoryKey{c.Type, lower(c.Name)}
			if _, ok := v.names[k]; !ok {
				v.names[k] = c.Name
			}
			v.states[k] = append(
				v.states[k],
				CategoryState{Space: id, Icon: c.Icon, Color: c.Color, IsDefault: c.IsDefault, Transactions: c.TransactionsCount},
			)
			if v.defaults[c.Type] == nil {
				v.defaults[c.Type] = map[int64]string{}
			}
			if c.IsDefault {
				v.defaults[c.Type][id] = c.Name
			}
		}
	}
	return v, nil
}

func diffCategories(
	ctx context.Context,
	dbs map[int64]store.DB,
	members []int64,
) ([]CategoryConflict, []DefaultConflict, error) {
	v, err := gatherCategories(ctx, dbs, members)
	if err != nil {
		return nil, nil, err
	}
	return v.conflicts(members), v.defaultConflicts(members), nil
}

// conflicts lists the categories that are missing from a space or look different in one.
func (v categoryView) conflicts(members []int64) []CategoryConflict {
	var conflicts []CategoryConflict
	for k, present := range v.states {
		var missing []int64
		for _, id := range members {
			if !slices.ContainsFunc(present, func(s CategoryState) bool { return s.Space == id }) {
				missing = append(missing, id)
			}
		}
		differs := false
		for _, s := range present[1:] {
			if !sameText(s.Icon, present[0].Icon) || !sameText(s.Color, present[0].Color) {
				differs = true
			}
		}
		if len(missing) > 0 || differs {
			conflicts = append(conflicts, CategoryConflict{Type: k.typ, Name: v.names[k], Present: present, Missing: missing})
		}
	}
	slices.SortFunc(conflicts, func(a, b CategoryConflict) int {
		return strings.Compare(a.Type+"\x00"+lower(a.Name), b.Type+"\x00"+lower(b.Name))
	})
	return conflicts
}

// defaultConflicts lists the types whose default category differs between spaces.
func (v categoryView) defaultConflicts(members []int64) []DefaultConflict {
	var out []DefaultConflict
	for typ, bySpace := range v.defaults {
		seen := map[string]bool{}
		full := map[int64]string{}
		for _, id := range members {
			full[id] = bySpace[id]
			seen[lower(bySpace[id])] = true
		}
		if len(seen) > 1 {
			out = append(out, DefaultConflict{Type: typ, Defaults: full})
		}
	}
	slices.SortFunc(out, func(a, b DefaultConflict) int { return strings.Compare(a.Type, b.Type) })
	return out
}

func diffTags(ctx context.Context, dbs map[int64]store.DB, members []int64) ([]TagConflict, error) {
	states := map[string][]TagState{}
	names := map[string]string{}
	for _, id := range members {
		list, err := (ledger.Tags{DB: dbs[id]}).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, t := range list {
			k := lower(t.Name)
			if _, ok := names[k]; !ok {
				names[k] = t.Name
			}
			states[k] = append(states[k], TagState{Space: id, Transactions: t.TransactionsCount})
		}
	}
	var out []TagConflict
	for k, present := range states {
		var missing []int64
		for _, id := range members {
			if !slices.ContainsFunc(present, func(s TagState) bool { return s.Space == id }) {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			out = append(out, TagConflict{Name: names[k], Present: present, Missing: missing})
		}
	}
	slices.SortFunc(out, func(a, b TagConflict) int { return strings.Compare(lower(a.Name), lower(b.Name)) })
	return out, nil
}

func decisionKey(kind Kind, typ, name string) string {
	return string(kind) + "\x00" + typ + "\x00" + lower(name)
}
