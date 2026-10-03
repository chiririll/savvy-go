package db

import (
	"context"
	"database/sql"
	"errors"
)

// ErrLossyRescale means moving an account to a currency with fewer decimals
// would drop non-zero digits from stored amounts.
var ErrLossyRescale = errors.New("amounts would lose precision in the new currency")

func pow10(n int) int64 {
	p := int64(1)
	for i := 0; i < n; i++ {
		p *= 10
	}
	return p
}

// RescaleAccountAmounts rewrites every stored minor-unit amount that is scaled
// by the account's currency decimals, keeping each value numerically equal.
// It runs inside tx and returns ErrLossyRescale (touching nothing) when going
// to fewer decimals would round away digits.
func RescaleAccountAmounts(ctx context.Context, tx *sql.Tx, accountID int64, oldDec, newDec int) error {
	if oldDec == newDec {
		return nil
	}
	mul, div := int64(1), int64(1)
	if newDec > oldDec {
		mul = pow10(newDec - oldDec)
	} else {
		div = pow10(oldDec - newDec)
	}
	if div > 1 {
		var lossy int64
		err := tx.QueryRowContext(ctx, `
			SELECT
				(SELECT COUNT(*) FROM transactions WHERE account_id = ? AND amount % ? != 0)
				+ (SELECT COUNT(*) FROM transactions WHERE to_account_id = ? AND to_amount % ? != 0)
				+ (SELECT COUNT(*) FROM recurring_transactions WHERE account_id = ? AND amount % ? != 0)
				+ (SELECT COUNT(*) FROM recurring_transactions WHERE to_account_id = ? AND to_amount % ? != 0)
				+ (SELECT COUNT(*) FROM transaction_items
					WHERE transaction_id IN (SELECT id FROM transactions WHERE account_id = ?)
					AND (price_per_unit % ? != 0 OR total_price % ? != 0))`,
			accountID, div, accountID, div, accountID, div, accountID, div, accountID, div, div).Scan(&lossy)
		if err != nil {
			return err
		}
		if lossy > 0 {
			return ErrLossyRescale
		}
	}
	steps := []struct {
		query string
		args  []any
	}{
		{`UPDATE transactions SET amount = amount * ? / ? WHERE account_id = ?`, []any{mul, div, accountID}},
		{`UPDATE transactions SET to_amount = to_amount * ? / ? WHERE to_account_id = ?`, []any{mul, div, accountID}},
		{`UPDATE recurring_transactions SET amount = amount * ? / ? WHERE account_id = ?`, []any{mul, div, accountID}},
		{`UPDATE recurring_transactions SET to_amount = to_amount * ? / ? WHERE to_account_id = ?`, []any{mul, div, accountID}},
		{`UPDATE transaction_items SET price_per_unit = price_per_unit * ? / ?, total_price = total_price * ? / ?
			WHERE transaction_id IN (SELECT id FROM transactions WHERE account_id = ?)`, []any{mul, div, mul, div, accountID}},
	}
	for _, s := range steps {
		if _, err := tx.ExecContext(ctx, s.query, s.args...); err != nil {
			return err
		}
	}
	return nil
}
