package ledger

import (
	"context"
)

// SimilarFilter selects transactions that moved Minor (its absolute value, in
// minor units) in the currency CurrencyID, dated in [From, To) as texts.
// AccountID narrows to one account when above 0. Skipped transactions never
// move money, so they are not similar to anything.
type SimilarFilter struct {
	CurrencyID int64
	AccountID  int64
	Minor      int64
	From, To   string
	Limit      int
}

// SimilarIDs returns the ids of the transactions with a payment that matches
// f, newest first.
func (r Repo) SimilarIDs(ctx context.Context, f SimilarFilter) ([]int64, error) {
	q := `SELECT DISTINCT t.id, t.date FROM transactions t
		JOIN payments p ON p.transaction_id = t.id
		JOIN accounts a ON a.id = p.account_id
		WHERE a.currency_id = ? AND ABS(p.amount) = ? AND t.date >= ? AND t.date < ? AND t.status <> 'skipped'`
	args := []any{f.CurrencyID, f.Minor, f.From, f.To}
	if f.AccountID > 0 {
		q += ` AND p.account_id = ?`
		args = append(args, f.AccountID)
	}
	q += ` ORDER BY t.date DESC, t.id DESC LIMIT ?`
	args = append(args, f.Limit)
	return r.ids(ctx, q, args...)
}

// IDsByExternalID returns the ids of the transactions with this external id
// under any source, oldest first.
func (r Repo) IDsByExternalID(ctx context.Context, externalID string, limit int) ([]int64, error) {
	return r.ids(ctx, `SELECT id, NULL FROM transactions WHERE external_id = ? ORDER BY id LIMIT ?`, externalID, limit)
}

// ids reads the first column of a query that selects two.
func (r Repo) ids(ctx context.Context, q string, args ...any) ([]int64, error) {
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		var ignored any
		if err := rows.Scan(&id, &ignored); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
