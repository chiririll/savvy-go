package seed

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	appdb "savvy-go/internal/db"
	"savvy-go/internal/domain"
	"savvy-go/internal/settings"
	"savvy-go/internal/store"
)

const demoSeededKey = "demo_seeded"

// Demo user keys, used to wire up spaces and memberships.
const (
	userAlex   = "alex"
	userJordan = "jordan"
	userSam    = "sam"
	userDemo   = "demo"
)

// Demo seeds demo users and several spaces when enabled and the server has
// never been demo-seeded (no users yet). Later starts are no-ops so first-boot
// matches Laravel's SEED_DEMO behavior.
//
// The spaces cover every feature of the spaces model:
//   - "Demo": the full workspace in USD with a EUR account; Alex administers it,
//     Jordan edits it and the demo user only views it.
//   - "Family Budget": Alex and Jordan administer it, the guest Sam views it.
//     It is linked to Demo.
//   - "Studio GmbH": a EUR-based space Alex administers, Jordan only views. It is
//     linked to Demo, so transfers between them cross currencies, and it is not
//     linked to Family Budget. It has open invitations.
func Demo(ctx context.Context, st store.Store, keys domain.KeyRing, enabled bool, loc *time.Location) error {
	if !enabled {
		return nil
	}
	if loc == nil {
		loc = time.UTC
	}
	server := settings.Store{DB: st.Server()}
	if server.Bool(ctx, demoSeededKey, false) {
		return nil
	}
	users := auth.Users{DB: st.Server()}
	n, err := users.Count(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if n > 0 {
		return nil
	}

	people, err := createDemoUsers(ctx, users)
	if err != nil {
		return err
	}
	spaces := domain.Spaces{Store: st}
	d := &demo{ctx: ctx, st: st, spaces: spaces, people: people, now: time.Now().In(loc)}

	d.demo, err = d.createSpace("Demo", userAlex, map[string]string{userJordan: domain.SpaceEditor, userDemo: domain.SpaceViewer})
	if err != nil {
		return err
	}
	d.family, err = d.createSpace("Family Budget", userAlex, map[string]string{userJordan: domain.SpaceAdmin, userSam: domain.SpaceViewer})
	if err != nil {
		return err
	}
	d.studio, err = d.createSpace("Studio GmbH", userAlex, map[string]string{userJordan: domain.SpaceViewer})
	if err != nil {
		return err
	}

	demoAccts, err := d.fill(d.demo, currencySet(usdCurrency, withRate(eurCurrency, "1.08")), func(db store.DB) (map[string]*domain.Account, error) {
		return seedWorkspace(ctx, db, loc)
	})
	if err != nil {
		return err
	}
	familyAccts, err := d.fill(d.family, currencySet(usdCurrency), func(db store.DB) (map[string]*domain.Account, error) {
		return seedMini(ctx, db, d.now, familySpace)
	})
	if err != nil {
		return err
	}
	studioAccts, err := d.fill(d.studio, currencySet(eurCurrency, withRate(usdCurrency, "0.9259")), func(db store.DB) (map[string]*domain.Account, error) {
		return seedMini(ctx, db, d.now, studioSpace)
	})
	if err != nil {
		return err
	}

	transfers := domain.Transfers{Spaces: spaces, Keys: keys}
	if err := d.linkAndTransfer(transfers, demoAccts, familyAccts, studioAccts); err != nil {
		return err
	}
	if err := d.invite(); err != nil {
		return err
	}
	if err := server.Set(ctx, demoSeededKey, true); err != nil {
		return fmt.Errorf("mark demo seeded: %w", err)
	}

	for _, sp := range []*domain.Space{d.demo, d.family, d.studio} {
		db, err := st.Space(ctx, sp.ID)
		if err != nil {
			return err
		}
		txs, _ := appdb.Q(db).CountAllTransactions(ctx)
		accts, _ := appdb.Q(db).CountAccounts(ctx)
		slog.Info("demo space seeded", "space", sp.Name, "transactions", txs, "accounts", accts)
	}
	return nil
}

func createDemoUsers(ctx context.Context, users auth.Users) (map[string]*auth.User, error) {
	out := map[string]*auth.User{}
	for _, u := range []struct {
		key, name, email, pass, role string
	}{
		{userAlex, "Alex Morgan", "admin@savvy.app", "password", auth.RoleAdmin},
		{userJordan, "Jordan Lee", "editor@savvy.app", "password", auth.RoleUser},
		{userSam, "Sam Rivera", "guest@savvy.app", "password", auth.RoleGuest},
		{userDemo, "Demo User", "demo@demo.com", "demo", auth.RoleUser},
	} {
		pass := u.pass
		created, err := users.Create(ctx, u.name, u.email, &pass, u.role)
		if err != nil {
			return nil, fmt.Errorf("user %s: %w", u.email, err)
		}
		out[u.key] = created
	}
	return out, nil
}

// demo carries what the steps of Demo share.
type demo struct {
	ctx    context.Context
	st     store.Store
	spaces domain.Spaces
	people map[string]*auth.User
	now    time.Time

	demo, family, studio *domain.Space
}

// createSpace creates a space owned by owner and adds the other members.
func (d *demo) createSpace(name, owner string, members map[string]string) (*domain.Space, error) {
	sp, err := d.spaces.Create(d.ctx, name, d.people[owner])
	if err != nil {
		return nil, fmt.Errorf("space %s: %w", name, err)
	}
	for key, role := range members {
		if err := d.spaces.SetMember(d.ctx, sp.ID, d.people[key], role); err != nil {
			return nil, fmt.Errorf("space %s member %s: %w", name, key, err)
		}
	}
	return sp, nil
}

// fill seeds a space's reference data and then its workspace.
func (d *demo) fill(sp *domain.Space, currencies []domain.Currency, workspace func(store.DB) (map[string]*domain.Account, error)) (map[string]*domain.Account, error) {
	db, err := d.st.Space(d.ctx, sp.ID)
	if err != nil {
		return nil, err
	}
	if err := seedReference(d.ctx, db, currencies); err != nil {
		return nil, fmt.Errorf("space %s: %w", sp.Name, err)
	}
	accts, err := workspace(db)
	if err != nil {
		return nil, fmt.Errorf("space %s: %w", sp.Name, err)
	}
	return accts, nil
}

// linkAndTransfer links Demo with the other two spaces and moves money across
// each link, including a transfer between currencies.
func (d *demo) linkAndTransfer(tr domain.Transfers, demoAccts, familyAccts, studioAccts map[string]*domain.Account) error {
	alex := d.people[userAlex]
	for _, other := range []*domain.Space{d.family, d.studio} {
		if err := tr.Link(d.ctx, alex, d.demo.ID, other.ID); err != nil {
			return fmt.Errorf("link %s: %w", other.Name, err)
		}
	}
	for _, t := range []struct {
		daysAgo              int
		from, to             *domain.Space
		fromAcct, toAcct     *domain.Account
		fromAmount, toAmount string
		description          string
	}{
		{40, d.demo, d.family, demoAccts["checking"], familyAccts["joint"], "900", "900", "Monthly contribution to the family account"},
		{12, d.demo, d.family, demoAccts["checking"], familyAccts["joint"], "900", "900", "Monthly contribution to the family account"},
		{25, d.family, d.demo, familyAccts["joint"], demoAccts["cash"], "150", "150", "Pocket money back"},
		// Across currencies: each side keeps its own currency and amount.
		{30, d.demo, d.studio, demoAccts["eur"], studioAccts["bank"], "1200", "1200", "Capital for the studio"},
		{9, d.studio, d.demo, studioAccts["bank"], demoAccts["checking"], "2000", "2160", "Owner's draw"},
	} {
		date := d.now.AddDate(0, 0, -t.daysAgo).Format("2006-01-02")
		desc := t.description
		if _, err := tr.Create(d.ctx, alex, domain.TransferInput{
			FromSpace: t.from.ID, FromAccount: t.fromAcct.ID, FromAmount: decimal.RequireFromString(t.fromAmount),
			ToSpace: t.to.ID, ToAccount: t.toAcct.ID, ToAmount: decimal.RequireFromString(t.toAmount),
			Date: date, Description: &desc,
		}); err != nil {
			return fmt.Errorf("transfer %q: %w", t.description, err)
		}
	}
	return nil
}

// invite leaves two open invitations in Studio GmbH: one bound to an email and
// one anyone with the link can use.
func (d *demo) invite() error {
	alex := d.people[userAlex]
	if _, err := d.spaces.Invite(d.ctx, d.studio.ID, alex, "newhire@example.com", domain.SpaceEditor); err != nil {
		return fmt.Errorf("invitation: %w", err)
	}
	if _, err := d.spaces.Invite(d.ctx, d.studio.ID, alex, "", domain.SpaceViewer); err != nil {
		return fmt.Errorf("invitation: %w", err)
	}
	return nil
}
