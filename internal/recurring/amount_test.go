package recurring

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/ledger"
	"savvy-go/internal/migrate"
	"savvy-go/internal/money"
	"savvy-go/internal/money/moneytest"
)

// A recurring transaction template with an amount that does not fit the
// currency is refused and stores nothing, in every currency scale.
func TestAmountsOutRangeRejectedStoreNothing(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		ctx := context.Background()
		sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = sqlDB.Close() })
		if err := migrate.Space.Up(ctx, sqlDB); err != nil {
			t.Fatal(err)
		}
		cur, err := (ledger.Currencies{DB: sqlDB}).Create(ctx, ledger.Currency{
			Code: c.Code, Name: c.Code, Symbol: c.Code, Decimals: c.Decimals, Rate: decimal.NewFromInt(1),
		})
		if err != nil {
			t.Fatal(err)
		}
		acct, err := (ledger.Accounts{DB: sqlDB}).Create(
			ctx,
			ledger.AccountInput{Name: "a", Type: "cash", CurrencyID: cur.ID, IsActive: true},
		)
		if err != nil {
			t.Fatal(err)
		}
		svc := Service{DB: sqlDB, Txs: ledger.Transactions{DB: sqlDB}}

		moneytest.ForEachAmount(t, moneytest.Invalid(c.Decimals), func(t *testing.T, a moneytest.Amount) {
			for _, in := range []decimal.Decimal{a.Input, a.Input.Neg()} {
				_, err := svc.Create(
					ctx,
					Input{Type: "expense", AccountID: acct.ID, Amount: in, Frequency: "monthly", StartDate: "2024-01-15"},
				)
				if !errors.Is(err, money.ErrOutOfRange) {
					t.Errorf("amount %s: err = %v, want ErrOutOfRange", in, err)
				}
			}
			var n int
			if err := sqlDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM recurring_transactions`).Scan(&n); err != nil || n != 0 {
				t.Errorf("%d recurring templates stored (%v)", n, err)
			}
		})
	})
}
