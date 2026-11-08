package ledger

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

// Why a transaction was found similar to a query.
const (
	MatchExternalID  = "external_id"
	MatchAmount      = "amount"
	MatchDate        = "date"
	MatchAccount     = "account"
	MatchDescription = "description"
)

const (
	defaultSimilarPerItem = 5
	maxSimilarPerItem     = 50
	maxSimilarDays        = 31
)

// SimilarQuery describes a transaction that may already be entered: what a
// bank line or a receipt says. Date and Amount find it by money; ExternalID
// finds it by key under any source. At least one of Date or ExternalID is
// needed. The amount is read in a currency, given by Currency (a code) or by
// the account.
type SimilarQuery struct {
	Key         string // echoed back by the caller; the ledger ignores it
	ExternalID  *string
	Date        string // YYYY-MM-DD
	Amount      decimal.Decimal
	Currency    string
	AccountID   int64
	Description *string
}

// SimilarOptions tune a search. ToleranceDays is how many days either side of
// the date still count (0: the same day only).
type SimilarOptions struct {
	ToleranceDays int
	MaxPerItem    int
}

// SimilarMatch is an entered transaction that resembles a query, and why.
type SimilarMatch struct {
	Transaction *Transaction
	Reasons     []string
}

// FindSimilar answers each query with the entered transactions that look like
// it, best first, in the order of the queries. The amount must equal a payment
// exactly in the query's currency, whatever the sign: 10 USD never matches
// 10 JPY, and 10.005 never matches 10.01. It reads only; nothing is stored.
func (s Transactions) FindSimilar(ctx context.Context, qs []SimilarQuery, opt SimilarOptions) ([][]SimilarMatch, error) {
	opt.MaxPerItem = clampDefault(opt.MaxPerItem, defaultSimilarPerItem, maxSimilarPerItem)
	if opt.ToleranceDays < 0 || opt.ToleranceDays > maxSimilarDays {
		return nil, fmt.Errorf("date tolerance must be between 0 and %d days", maxSimilarDays)
	}
	out := make([][]SimilarMatch, len(qs))
	for i, q := range qs {
		found, err := s.findSimilar(ctx, q, opt)
		if err != nil {
			return nil, fmt.Errorf("item %d: %w", i+1, err)
		}
		out[i] = found
	}
	return out, nil
}

func clampDefault(v, def, upper int) int {
	switch {
	case v <= 0:
		return def
	case v > upper:
		return upper
	}
	return v
}

// similarSearch is a query read and checked: the unit and day window of the
// money search, or no unit when only the external id is searched.
type similarSearch struct {
	unit       money.Unit
	minor      int64
	day        time.Time
	byAmount   bool
	externalID string
}

func (s Transactions) readSimilar(ctx context.Context, q SimilarQuery) (similarSearch, error) {
	var out similarSearch
	if q.ExternalID != nil {
		out.externalID = strings.TrimSpace(*q.ExternalID)
	}
	if q.Date == "" {
		if out.externalID == "" {
			return out, fmt.Errorf("date or external_id is needed")
		}
		return out, nil
	}
	day, err := time.Parse("2006-01-02", q.Date)
	if err != nil {
		return out, fmt.Errorf("date must be YYYY-MM-DD")
	}
	if q.Amount.IsZero() {
		return out, fmt.Errorf("amount must not be zero")
	}
	cur, err := s.similarCurrency(ctx, q)
	if err != nil {
		return out, err
	}
	// Not rounded: an amount finer than the currency keeps is not any stored amount.
	if !q.Amount.Round(int32(cur.Decimals)).Equal(q.Amount) {
		return out, nil
	}
	m, err := money.FromInput(q.Amount.Abs(), cur.Unit())
	if err != nil {
		return out, fmt.Errorf("amount: %w", err)
	}
	out.unit, out.minor, out.day, out.byAmount = cur.Unit(), m.Minor(), day, true
	return out, nil
}

// similarCurrency is the currency of the query: its account's, or the code.
func (s Transactions) similarCurrency(ctx context.Context, q SimilarQuery) (*Currency, error) {
	curs := Currencies{DB: s.DB}
	if q.AccountID > 0 {
		acc, err := (Accounts{DB: s.DB}).Bare(ctx, q.AccountID)
		if err != nil {
			return nil, err
		}
		if acc == nil {
			return nil, fmt.Errorf("unknown account %d", q.AccountID)
		}
		c, err := curs.ByID(ctx, acc.CurrencyID)
		if err != nil || c == nil {
			return nil, fmt.Errorf("account %d has no currency: %v", q.AccountID, err)
		}
		if q.Currency != "" && !strings.EqualFold(q.Currency, c.Code) {
			return nil, fmt.Errorf("account %d is in %s, not %s", q.AccountID, c.Code, q.Currency)
		}
		return c, nil
	}
	if q.Currency == "" {
		return nil, fmt.Errorf("currency or account_id is needed to read the amount")
	}
	c, err := curs.ByCode(ctx, q.Currency)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("unknown currency %q", q.Currency)
	}
	return c, nil
}

func (s Transactions) findSimilar(ctx context.Context, q SimilarQuery, opt SimilarOptions) ([]SimilarMatch, error) {
	sr, err := s.readSimilar(ctx, q)
	if err != nil {
		return nil, err
	}
	r := s.repo()
	var byKey []int64
	if sr.externalID != "" {
		if byKey, err = r.IDsByExternalID(ctx, sr.externalID, maxSimilarPerItem); err != nil {
			return nil, err
		}
	}
	var byMoney []int64
	if sr.byAmount {
		days := time.Duration(opt.ToleranceDays) * 24 * time.Hour
		byMoney, err = r.SimilarIDs(ctx, repo.SimilarFilter{
			CurrencyID: sr.unit.ID, AccountID: q.AccountID, Minor: sr.minor,
			From: sr.day.Add(-days).Format("2006-01-02"), To: sr.day.Add(days + 24*time.Hour).Format("2006-01-02"),
			Limit: maxSimilarPerItem,
		})
		if err != nil {
			return nil, err
		}
	}
	var out []SimilarMatch
	seen := map[int64]bool{}
	for _, id := range slices.Concat(byKey, byMoney) {
		if seen[id] {
			continue
		}
		seen[id] = true
		tx, err := s.ByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if tx != nil {
			out = append(out, SimilarMatch{Transaction: tx, Reasons: similarReasons(tx, q, sr, opt, slices.Contains(byKey, id))})
		}
	}
	sortSimilar(out, sr)
	return out[:min(len(out), opt.MaxPerItem)], nil
}

// similarReasons says what a found transaction has in common with the query.
func similarReasons(tx *Transaction, q SimilarQuery, sr similarSearch, opt SimilarOptions, byKey bool) []string {
	var why []string
	if byKey {
		why = append(why, MatchExternalID)
	}
	if sr.byAmount {
		for _, p := range tx.Payments {
			if p.Amount.Unit().ID == sr.unit.ID && p.Amount.Abs().Minor() == sr.minor {
				why = append(why, MatchAmount)
				break
			}
		}
		if d, ok := dayDistance(tx, sr.day); ok && d <= opt.ToleranceDays {
			why = append(why, MatchDate)
		}
		if q.AccountID > 0 && slices.ContainsFunc(tx.Payments, func(p Payment) bool { return p.AccountID == q.AccountID }) {
			why = append(why, MatchAccount)
		}
	}
	if sameDescription(tx.Description, q.Description) {
		why = append(why, MatchDescription)
	}
	return why
}

// sortSimilar puts the best first: the key, then the description, then the
// nearest day, then the newest.
func sortSimilar(ms []SimilarMatch, sr similarSearch) {
	rank := func(m SimilarMatch) (key, desc bool, dist int) {
		d, ok := dayDistance(m.Transaction, sr.day)
		if !ok {
			d = 1 << 30
		}
		return slices.Contains(m.Reasons, MatchExternalID), slices.Contains(m.Reasons, MatchDescription), d
	}
	slices.SortStableFunc(ms, func(a, b SimilarMatch) int {
		ak, ad, adist := rank(a)
		bk, bd, bdist := rank(b)
		switch {
		case ak != bk:
			return boolRank(bk) - boolRank(ak)
		case ad != bd:
			return boolRank(bd) - boolRank(ad)
		case adist != bdist:
			return adist - bdist
		}
		return int(b.Transaction.ID - a.Transaction.ID)
	})
}

func boolRank(b bool) int {
	if b {
		return 1
	}
	return 0
}

// dayDistance is how many days tx is dated from day; false for an undated or
// unreadable date, or when no day is searched.
func dayDistance(tx *Transaction, day time.Time) (int, bool) {
	if tx.Date == nil || len(*tx.Date) < 10 || day.IsZero() {
		return 0, false
	}
	d, err := time.Parse("2006-01-02", (*tx.Date)[:10])
	if err != nil {
		return 0, false
	}
	diff := int(d.Sub(day).Hours() / 24)
	if diff < 0 {
		diff = -diff
	}
	return diff, true
}

// sameDescription compares descriptions without regard to case and spacing.
func sameDescription(a, b *string) bool {
	norm := func(s *string) string {
		if s == nil {
			return ""
		}
		return strings.ToLower(strings.Join(strings.Fields(*s), " "))
	}
	x, y := norm(a), norm(b)
	return x != "" && x == y
}
