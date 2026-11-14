package seed

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"savvy-go/internal/auth"
	automationrepo "savvy-go/internal/db/repo/automation"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/ledger"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// Space keys, used to refer to a seeded space.
const (
	spaceDemo   = "demo"
	spaceFamily = "family"
	spaceStudio = "studio"
)

// workspaceFunc fills a space whose reference data exists and returns its
// accounts by key.
type workspaceFunc func(ctx context.Context, db store.DB, now time.Time, months int) (map[string]*ledger.Account, error)

// spaceSpec is one seeded space: who belongs to it, what it is priced in and
// what fills it. The owner is an admin of the space.
type spaceSpec struct {
	key, name  string
	emoji      string
	color      string // #rrggbb
	owner      string
	members    map[string]string // user key -> role
	currencies []ledger.Currency
	// own is the categories and tags only this space has, on top of the presets.
	own       ownReference
	workspace workspaceFunc
}

func (s spaceSpec) look() spaces.Look { return spaces.Look{Emoji: &s.emoji, Color: &s.color} }

func miniWorkspace(spec miniSpace) workspaceFunc {
	return func(ctx context.Context, db store.DB, now time.Time, _ int) (map[string]*ledger.Account, error) {
		return seedMini(ctx, db, now, spec)
	}
}

// demoSpaces are the seeded spaces, in the order they are created.
var demoSpaces = []spaceSpec{
	{
		key: spaceDemo, name: "Demo", emoji: "🏠", color: "#3b82f6", owner: userAlex,
		members:    map[string]string{userJordan: spaces.RoleEditor},
		currencies: currencySet(usdCurrency, withRate(eurCurrency, "1.08")),
		own: ownReference{
			categories: []Category{{"Pets", "expense", "🐾", "#a3e635", false}},
			tags:       []string{"Weekend"},
		},
		workspace: seedWorkspace,
	},
	{
		key: spaceFamily, name: "Family Budget", emoji: "👨‍👩‍👧", color: "#22c55e", owner: userAlex,
		members:    map[string]string{userJordan: spaces.RoleAdmin, userSam: spaces.RoleViewer},
		currencies: currencySet(usdCurrency),
		own: ownReference{
			categories: []Category{{"School & kids", "expense", "🎒", "#facc15", false}},
			tags:       []string{"Kids"},
		},
		workspace: miniWorkspace(familySpace),
	},
	{
		key: spaceStudio, name: "Studio GmbH", emoji: "🎨", color: "#f97316", owner: userAlex,
		members:    map[string]string{userJordan: spaces.RoleViewer},
		currencies: currencySet(eurCurrency, withRate(usdCurrency, "0.9259")),
		own:        studioReference,
		workspace:  miniWorkspace(studioSpace),
	},
}

// seededSpace is a space Demo created and the accounts it filled it with.
type seededSpace struct {
	*spaces.Space
	accounts map[string]*ledger.Account
}

// demo carries what the steps of Demo share.
type demo struct {
	ctx    context.Context
	st     store.Store
	spaces spaces.Service
	people map[string]*auth.User
	now    time.Time
	// months is how many months of history the Demo space gets.
	months int
	// nameSuffix is added to the space names, to tell the copies apart.
	nameSuffix string

	manifest *Manifest
	byKey    map[string]*seededSpace
	ordered  []*seededSpace
}

// space returns a seeded space by its key.
func (d *demo) space(key string) *seededSpace { return d.byKey[key] }

// createSpaces creates every space of demoSpaces and fills it.
func (d *demo) createSpaces() error {
	d.byKey = map[string]*seededSpace{}
	for _, spec := range demoSpaces {
		sp, err := d.createSpace(spec)
		if err != nil {
			return err
		}
		var accts map[string]*ledger.Account
		err = timed("seed space", func() (err error) {
			accts, err = d.fill(sp, spec)
			return err
		}, "space", sp.Name, "id", sp.ID, "members", len(spec.members)+1)
		if err != nil {
			return err
		}
		s := &seededSpace{Space: sp, accounts: accts}
		d.byKey[spec.key] = s
		d.ordered = append(d.ordered, s)
	}
	return nil
}

// createSpace creates a space owned by the spec's owner and adds the other
// members.
func (d *demo) createSpace(spec spaceSpec) (*spaces.Space, error) {
	sp, err := d.spaces.CreateWith(d.ctx, spec.name+d.nameSuffix, d.people[spec.owner], spec.look())
	if err != nil {
		return nil, fmt.Errorf("space %s: %w", spec.name, err)
	}
	roles := map[string]string{spec.owner: spaces.RoleAdmin}
	for key, role := range spec.members {
		if err := d.spaces.SetMember(d.ctx, sp.ID, d.people[key], role); err != nil {
			return nil, fmt.Errorf("space %s member %s: %w", spec.name, key, err)
		}
		roles[key] = role
	}
	d.manifest.Spaces = append(d.manifest.Spaces, ManifestSpace{
		ID: sp.ID, Name: spec.name + d.nameSuffix, Emoji: spec.emoji, Color: spec.color, Members: roles,
	})
	return sp, nil
}

// fill seeds a space's reference data and then its workspace.
func (d *demo) fill(sp *spaces.Space, spec spaceSpec) (map[string]*ledger.Account, error) {
	db, err := d.st.Space(d.ctx, sp.ID)
	if err != nil {
		return nil, err
	}
	if err := seedReference(d.ctx, db, spec.currencies, !spec.own.replacesPresets); err != nil {
		return nil, fmt.Errorf("space %s: %w", sp.Name, err)
	}
	if err := seedOwn(d.ctx, db, spec.own); err != nil {
		return nil, fmt.Errorf("space %s: %w", sp.Name, err)
	}
	accts, err := spec.workspace(d.ctx, db, d.now, d.months)
	if err != nil {
		return nil, fmt.Errorf("space %s: %w", sp.Name, err)
	}
	return accts, nil
}

// recordAutomations notes the automation rules of each space in the manifest,
// for the pages that address a rule by id.
func (d *demo) recordAutomations() error {
	for _, made := range d.ordered {
		sp := d.manifestSpace(made.ID)
		db, err := d.st.Space(d.ctx, sp.ID)
		if err != nil {
			return err
		}
		rules, err := automationrepo.Repo{DB: db}.List(d.ctx, automationrepo.Filter{})
		if err != nil {
			return fmt.Errorf("space %s automation rules: %w", sp.Name, err)
		}
		for _, r := range rules {
			sp.Automations = append(sp.Automations, r.ID)
		}
	}
	return nil
}

// manifestSpace returns the manifest entry of the space with id.
func (d *demo) manifestSpace(id int64) *ManifestSpace {
	for i := range d.manifest.Spaces {
		if d.manifest.Spaces[i].ID == id {
			return &d.manifest.Spaces[i]
		}
	}
	return nil
}

func (d *demo) logSeeded() error {
	for _, sp := range d.ordered {
		db, err := d.st.Space(d.ctx, sp.ID)
		if err != nil {
			return err
		}
		repo := ledgerrepo.Repo{DB: db}
		txs, accts := repo.CountTransactions(d.ctx), repo.CountAccounts(d.ctx)
		slog.Info("demo space seeded", "space", sp.Name, "transactions", txs, "accounts", accts)
	}
	return nil
}
