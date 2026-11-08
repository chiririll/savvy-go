package reports

import (
	"sort"

	"savvy-go/internal/ledger"
	"savvy-go/internal/money"
)

// catGroup is a top-level category with what sits under it. Total is the whole
// group: the parent's own lines plus its children's.
type catGroup struct {
	catTotal
	// Parts are the children, largest first, with the parent's own lines
	// (under the parent's id) among them; empty for a category without
	// children in the period.
	Parts []catTotal
}

// byParent folds each category into its top-level parent, largest group first.
// A parent with no lines of its own still heads its children's group; one that
// can no longer be found leaves the child at the top level.
func (q report) byParent(cats []catTotal) []catGroup {
	parents := q.categoriesByID(cats)
	groups := []*catGroup{}
	idx := map[int64]*catGroup{}
	group := func(head catTotal) *catGroup {
		if g, ok := idx[head.ID]; ok {
			return g
		}
		head.ParentID, head.Total = 0, q.zero()
		g := &catGroup{catTotal: head}
		idx[head.ID] = g
		groups = append(groups, g)
		return g
	}
	for _, c := range cats {
		var g *catGroup
		if p, ok := parents[c.ParentID]; ok && c.ParentID != 0 {
			g = group(catTotal{
				ID: p.ID, Name: p.Name,
				Icon:  coalesce(strOr(p.Icon), "circle"),
				Color: coalesce(strOr(p.Color), "#64748b"),
			})
		} else {
			c.ParentID = 0
			g = group(c)
		}
		g.Total = g.Total.Add(c.Total)
		g.Parts = append(g.Parts, c)
	}
	out := make([]catGroup, 0, len(groups))
	for _, g := range groups {
		// A lone top-level category is not a group of one.
		if len(g.Parts) == 1 && g.Parts[0].ID == g.ID {
			g.Parts = nil
		}
		sortByTotal(g.Parts)
		out = append(out, *g)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Total.Cmp(out[j].Total) > 0 })
	return out
}

// groupTotals is each top-level group's total by its id.
func (q report) groupTotals(cats []catTotal) map[int64]money.Money {
	out := map[int64]money.Money{}
	for _, g := range q.byParent(cats) {
		out[g.ID] = g.Total
	}
	return out
}

func sortByTotal(cats []catTotal) {
	sort.SliceStable(cats, func(i, j int) bool { return cats[i].Total.Cmp(cats[j].Total) > 0 })
}

// categoriesByID loads the categories the totals point to as parents; nil when
// none has a parent.
func (q report) categoriesByID(cats []catTotal) map[int64]ledger.Category {
	need := false
	for _, c := range cats {
		need = need || c.ParentID != 0
	}
	if !need {
		return nil
	}
	all, err := (ledger.Categories{DB: q.DB}).All(q.ctx, "")
	if err != nil {
		return nil
	}
	out := make(map[int64]ledger.Category, len(all))
	for _, c := range all {
		out[c.ID] = c
	}
	return out
}
