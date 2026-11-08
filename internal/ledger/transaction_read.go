package ledger

import (
	"context"
	"slices"
	"strings"

	"savvy-go/internal/db/filter"
	repo "savvy-go/internal/db/repo/ledger"
)

// Reading transactions.

func (s Transactions) ByID(ctx context.Context, id int64) (*Transaction, error) {
	return s.byID(ctx, id, false)
}

// byID is ByID; bare leaves out the account balances and category counts.
func (s Transactions) byID(ctx context.Context, id int64, bare bool) (*Transaction, error) {
	r, err := s.repo().Transaction(ctx, id)
	if r == nil || err != nil {
		return nil, err
	}
	rel := NewRelated(ctx, s.DB)
	rel.bare = bare
	list, err := s.hydrateWith(ctx, []repo.Tx{*r}, rel)
	if err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (s Transactions) Filtered(ctx context.Context, f filter.TxFilter, page, perPage int) ([]Transaction, int, error) {
	if perPage <= 0 {
		perPage = 25
	}
	if page <= 0 {
		page = 1
	}
	rows, total, err := s.repo().Filtered(ctx, f, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.hydrate(ctx, rows)
	return list, total, err
}

// FilteredMore is Filtered without the count of all matches: whether there is
// a page after this one instead.
func (s Transactions) FilteredMore(ctx context.Context, f filter.TxFilter, page, perPage int) ([]Transaction, bool, error) {
	if perPage <= 0 {
		perPage = 25
	}
	if page <= 0 {
		page = 1
	}
	rows, more, err := s.repo().FilteredMore(ctx, f, perPage, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	list, err := s.hydrate(ctx, rows)
	return list, more, err
}

// DayTotals nets the transactions matching f per day and currency (see the
// package-level DayTotals), newest day first. Unlike a page of the list it
// covers every transaction of those days, so a day split across pages gets its
// whole total. Bound f by Dates or a date range: it reads every match.
func (s Transactions) DayTotals(ctx context.Context, f filter.TxFilter) ([]DayTotal, error) {
	const batch = 500
	var all []Transaction
	for page := 1; ; page++ {
		list, more, err := s.FilteredMore(ctx, f, page, batch)
		if err != nil {
			return nil, err
		}
		all = append(all, list...)
		if !more {
			break
		}
	}
	days := DayTotals(all)
	slices.SortFunc(days, func(a, b DayTotal) int { return strings.Compare(b.Date, a.Date) })
	return days, nil
}
