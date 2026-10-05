package legacy

import (
	"errors"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
	"savvy-go/internal/money/moneytest"
)

// Laravel stored money as REAL or NUMERIC in major units; the importer must
// land on exactly the minor units a Go-made row would have, in every currency
// scale, for every kind of amount.
func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestFixMoneyConvertsEveryCurrencyAndAmount(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		sc := &scales{
			currencyDec: map[int64]int{1: c.Decimals},
			accountCur:  map[int64]int64{1: 1},
			txAccount:   map[int64]int64{1: 1},
			base:        1,
		}
		forms := map[string]func(a moneytest.Amount) any{
			"float":  func(a moneytest.Amount) any { f, _ := a.Input.Float64(); return f },
			"string": func(a moneytest.Amount) any { return a.Input.String() },
		}
		for form, in := range forms {
			t.Run(form, func(t *testing.T) {
				moneytest.ForEachAmount(t, moneytest.Amounts(c.Decimals), func(t *testing.T, a moneytest.Amount) {
					rows := []struct {
						table string
						cols  []string
						vals  []any
					}{
						{"accounts", []string{"currency_id", "initial_balance", "target_amount"}, []any{int64(1), in(a), in(a)}},
						{"transactions", []string{"account_id", "amount", "to_amount"}, []any{int64(1), in(a), in(a)}},
						{"recurring_transactions", []string{"account_id", "amount", "to_amount"}, []any{int64(1), in(a), nil}},
						{"transaction_items", []string{"transaction_id", "price_per_unit", "total_price"}, []any{int64(1), in(a), in(a)}},
						{"budgets", []string{"currency_id", "amount"}, []any{int64(1), in(a)}},
					}
					for _, r := range rows {
						err := sc.fixMoney(r.table, r.cols, r.vals)
						if !a.Valid {
							if !errors.Is(err, money.ErrOutOfRange) || !strings.Contains(err.Error(), r.table) {
								t.Fatalf("%s: err = %v, want an ErrOutOfRange naming the table", r.table, err)
							}
							continue
						}
						if err != nil {
							t.Fatalf("%s: %v", r.table, err)
						}
						for i, col := range r.cols {
							if col == "currency_id" || col == "account_id" || col == "transaction_id" || r.vals[i] == nil {
								continue
							}
							if got, ok := r.vals[i].(int64); !ok || got != a.Minor {
								t.Fatalf("%s.%s = %v, want %d", r.table, col, r.vals[i], a.Minor)
							}
						}
					}
				})
			})
		}
	})
}

// A transfer's destination amount is scaled by the destination account's
// currency, not the source's.
func TestFixMoneyScalesTransferByEachAccountsCurrency(t *testing.T) {
	moneytest.ForEachPair(t, func(t *testing.T, from, to moneytest.Currency) {
		sc := &scales{
			currencyDec: map[int64]int{1: from.Decimals, 2: to.Decimals},
			accountCur:  map[int64]int64{1: 1, 2: 2},
			base:        1,
		}
		cols := []string{"account_id", "to_account_id", "amount", "to_amount"}
		vals := []any{int64(1), int64(2), "12.345678901234", "98.765432109876"}
		if err := sc.fixMoney("transactions", cols, vals); err != nil {
			t.Fatal(err)
		}
		wantFrom := money.FromDecimal(dec("12.345678901234"), money.Unit{Decimals: from.Decimals}).Minor()
		wantTo := money.FromDecimal(dec("98.765432109876"), money.Unit{Decimals: to.Decimals}).Minor()
		if vals[2] != wantFrom || vals[3] != wantTo {
			t.Fatalf("amount %v / to_amount %v, want %d / %d", vals[2], vals[3], wantFrom, wantTo)
		}
	})
}
