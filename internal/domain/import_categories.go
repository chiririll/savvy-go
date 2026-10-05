package domain

import (
	"context"
	"database/sql"
	"savvy-go/internal/store"
	"sort"
	"strings"
)

// importCategory is a distinct category name found in the imported file.
type importCategory struct {
	Name    string
	Type    string // income or expense: whichever the rows using it mostly are
	Count   int
	MatchID *int64 // existing category with the same name and type, if any
}

// collectImportCategories lists the distinct category names of the valid rows,
// most used first. Names differing only in case are merged.
func collectImportCategories(rows [][]string, mapping, options map[string]any) []*importCategory {
	if _, ok := mappingIndex(mapping, "category"); !ok {
		return nil
	}
	byKey := map[string]*importCategory{}
	types := map[string]map[string]int{}
	for i, row := range rows {
		res := processImportRow(row, mapping, options, i+1)
		if res.err != "" || res.category == "" {
			continue
		}
		key := strings.ToLower(res.category)
		c := byKey[key]
		if c == nil {
			c = &importCategory{Name: res.category}
			byKey[key] = c
			types[key] = map[string]int{}
		}
		c.Count++
		types[key][res.typ]++
	}
	out := make([]*importCategory, 0, len(byKey))
	for key, c := range byKey {
		c.Type = "expense"
		if types[key]["income"] > types[key]["expense"] {
			c.Type = "income"
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func existingCategoryIndex(ctx context.Context, db store.DB) (map[string]int64, error) {
	all, err := Categories{DB: db}.All(ctx, "")
	if err != nil {
		return nil, err
	}
	index := make(map[string]int64, len(all))
	for _, c := range all {
		index[categoryKey(c.Name, c.Type)] = c.ID
	}
	return index, nil
}

func categoryKey(name, typ string) string {
	return strings.ToLower(strings.TrimSpace(name)) + "|" + typ
}

// matchImportCategories fills MatchID for categories that already exist.
func (s Imports) matchImportCategories(ctx context.Context, cats []*importCategory) error {
	if len(cats) == 0 {
		return nil
	}
	index, err := existingCategoryIndex(ctx, s.DB)
	if err != nil {
		return err
	}
	for _, c := range cats {
		if id, ok := index[categoryKey(c.Name, c.Type)]; ok {
			id := id
			c.MatchID = &id
		}
	}
	return nil
}

// categoryResolver turns category names from the file into category ids using
// the user's choices (options.category_map): an existing id, "create" or "skip".
// Names without a choice use an existing category of the same name, otherwise
// are created when create_missing_categories is on.
type categoryResolver struct {
	ctx           context.Context
	db            store.DB
	cats          map[string]*importCategory
	choices       map[string]any
	createMissing bool
	existing      map[string]int64
	cache         map[string]sql.NullInt64
	created       []string
}

func (s Imports) newCategoryResolver(ctx context.Context, cats []*importCategory, options map[string]any) (*categoryResolver, error) {
	existing, err := existingCategoryIndex(ctx, s.DB)
	if err != nil {
		return nil, err
	}
	r := &categoryResolver{
		ctx: ctx, db: s.DB, existing: existing,
		cats: map[string]*importCategory{}, choices: map[string]any{}, cache: map[string]sql.NullInt64{},
		created: []string{},
	}
	for _, c := range cats {
		r.cats[strings.ToLower(c.Name)] = c
	}
	if m, ok := options["category_map"].(map[string]any); ok {
		for name, choice := range m {
			r.choices[strings.ToLower(strings.TrimSpace(name))] = choice
		}
	}
	if v, ok := options["create_missing_categories"].(bool); ok {
		r.createMissing = v
	}
	return r, nil
}

func (r *categoryResolver) resolve(name string) sql.NullInt64 {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return sql.NullInt64{}
	}
	if v, ok := r.cache[key]; ok {
		return v
	}
	v := r.lookup(key, name)
	r.cache[key] = v
	return v
}

func (r *categoryResolver) lookup(key, name string) sql.NullInt64 {
	typ := "expense"
	if c := r.cats[key]; c != nil {
		typ = c.Type
		name = c.Name
	}
	choice, chosen := r.choices[key]
	if chosen {
		if id, ok := asInt64(choice); ok {
			if c, err := (Categories{DB: r.db}).ByID(r.ctx, id); err == nil && c != nil {
				return sql.NullInt64{Int64: id, Valid: true}
			}
			return sql.NullInt64{}
		}
		if s, _ := choice.(string); s != "create" {
			return sql.NullInt64{} // "skip" or null: leave the transaction uncategorized
		}
	}
	if !chosen {
		if id, ok := r.existing[categoryKey(name, typ)]; ok {
			return sql.NullInt64{Int64: id, Valid: true}
		}
		if !r.createMissing {
			return sql.NullInt64{}
		}
	}
	if id, ok := r.existing[categoryKey(name, typ)]; ok { // "create" for a name that already exists
		return sql.NullInt64{Int64: id, Valid: true}
	}
	c, err := (Categories{DB: r.db}).Create(r.ctx, Category{Name: name, Type: typ})
	if err != nil || c == nil {
		return sql.NullInt64{}
	}
	r.existing[categoryKey(name, typ)] = c.ID
	r.created = append(r.created, name)
	return sql.NullInt64{Int64: c.ID, Valid: true}
}
