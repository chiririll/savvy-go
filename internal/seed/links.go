package seed

import (
	"fmt"

	"github.com/shopspring/decimal"

	"savvy-go/internal/spaces"
	"savvy-go/internal/transfers"
)

// transferSpec is one signed transfer between two linked spaces, by space key
// and account key. Each side keeps its own currency and amount.
type transferSpec struct {
	daysAgo              int
	from, fromAcct       string
	to, toAcct           string
	fromAmount, toAmount string
	description          string
}

const monthlyContribution = "Monthly contribution to the family account"

var demoTransfers = []transferSpec{
	{40, spaceDemo, "checking", spaceFamily, "joint", "900", "900", monthlyContribution},
	{12, spaceDemo, "checking", spaceFamily, "joint", "900", "900", monthlyContribution},
	{25, spaceFamily, "joint", spaceDemo, "cash", "150", "150", "Pocket money back"},
	// Across currencies.
	{30, spaceDemo, "eur", spaceStudio, "bank", "1200", "1200", "Capital for the studio"},
	{9, spaceStudio, "bank", spaceDemo, "checking", "2000", "2160", "Owner's draw"},
	{70, spaceDemo, "checking", spaceFamily, "joint", "900", "900", monthlyContribution},
	{100, spaceDemo, "checking", spaceFamily, "joint", "900", "900", monthlyContribution},
	{50, spaceStudio, "bank", spaceDemo, "eur", "600", "600", "Shared software licences"},
	{21, spaceStudio, "usd", spaceDemo, "checking", "600", "600", "Contractor refund"},
	{18, spaceDemo, "checking", spaceStudio, "usd", "1500", "1500", "Client prepayment"},
	{7, spaceFamily, "wallet", spaceDemo, "cash", "40", "40", "Cash for the trip"},
	// Recent, so the latest activity of the spaces shows them.
	{5, spaceStudio, "bank", spaceDemo, "eur", "800", "800", "Shared software licences"},
	{1, spaceDemo, "checking", spaceFamily, "joint", "450", "450", "Vacation fund top-up"},
	{2, spaceDemo, "checking", spaceStudio, "bank", "540", "500", "Studio retainer"},
}

// linkTransfer links Demo with the other two spaces and moves money across
// each link, including a transfer between currencies.
func (d *demo) linkTransfer(tr transfers.Service) error {
	alex := d.people[userAlex]
	home := d.space(spaceDemo)
	for _, key := range []string{spaceFamily, spaceStudio} {
		other := d.space(key)
		if err := tr.Link(d.ctx, alex, home.ID, other.ID); err != nil {
			return fmt.Errorf("link %s: %w", other.Name, err)
		}
	}
	for _, t := range demoTransfers {
		from, to := d.space(t.from), d.space(t.to)
		date := d.now.AddDate(0, 0, -t.daysAgo).Format("2006-01-02")
		desc := t.description
		if _, err := tr.Create(d.ctx, alex, transfers.Input{
			FromSpace: from.ID, FromAccount: from.accounts[t.fromAcct].ID,
			FromAmount: decimal.RequireFromString(t.fromAmount),
			ToSpace:    to.ID, ToAccount: to.accounts[t.toAcct].ID,
			ToAmount: decimal.RequireFromString(t.toAmount),
			Date:     date, Description: &desc,
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
	studio := d.space(spaceStudio)
	for _, inv := range []struct{ email, role string }{
		{"newhire@example.com", spaces.RoleEditor},
		{"", spaces.RoleViewer},
	} {
		token, _, err := d.spaces.Invite(d.ctx, studio.ID, alex, inv.email, "", inv.role)
		if err != nil {
			return fmt.Errorf("invitation: %w", err)
		}
		d.manifest.Invitations = append(d.manifest.Invitations, ManifestInvitation{
			SpaceID: studio.ID, Email: inv.email, Role: inv.role, Token: token,
		})
	}
	return nil
}
