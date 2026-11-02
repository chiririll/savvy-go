package ledger

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
	"savvy-go/internal/money"
	"savvy-go/internal/money/moneytest"
)

// matrixEnv is a database holding every moneytest currency, rates expressed
// against base, and an empty cash account in each.
type matrixEnv struct {
	ctx   context.Context
	db    *sql.DB
	base  moneytest.Currency
	curs  map[string]Currency
	accts map[string]Account
	txs   Transactions
}

func newMatrixEnv(t *testing.T, base moneytest.Currency) *matrixEnv {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	ctx := context.Background()
	if err := migrate.Space.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	e := &matrixEnv{
		ctx:   ctx,
		db:    sqlDB,
		base:  base,
		curs:  map[string]Currency{},
		accts: map[string]Account{},
		txs:   Transactions{DB: sqlDB},
	}
	add := func(c moneytest.Currency) {
		created, err := (Currencies{DB: sqlDB}).Create(ctx, Currency{
			Code: c.Code, Name: c.Code, Symbol: c.Code, Decimals: c.Decimals, Rate: c.RateIn(base),
		})
		if err != nil {
			t.Fatalf("create %v: %v", c, err)
		}
		e.curs[c.Code] = *created
		acct, err := (Accounts{DB: sqlDB}).Create(
			ctx,
			AccountInput{Name: c.Code, Type: "cash", CurrencyID: created.ID, IsActive: true},
		)
		if err != nil {
			t.Fatalf("account %v: %v", c, err)
		}
		e.accts[c.Code] = *acct
	}
	add(base) // first, so it becomes the base currency
	for _, c := range moneytest.Currencies {
		if c.Code != base.Code {
			add(c)
		}
	}
	return e
}

// tx records a confirmed transaction of the given type and amount on c's account.
func (e *matrixEnv) tx(t *testing.T, typ string, c moneytest.Currency, amount decimal.Decimal) (*Transaction, error) {
	t.Helper()
	date := "2024-01-15"
	in := simple(typ, e.accts[c.Code].ID, amount.String())
	in.Date = &date
	return e.txs.Create(e.ctx, in)
}

func (e *matrixEnv) balance(t *testing.T, c moneytest.Currency) money.Money {
	t.Helper()
	a, err := (Accounts{DB: e.db}).ByID(e.ctx, e.accts[c.Code].ID)
	if err != nil || a == nil {
		t.Fatalf("account %v: %v", c, err)
	}
	return a.Balance
}

func (e *matrixEnv) count(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (e *matrixEnv) rawAmount(t *testing.T, id int64) int64 {
	t.Helper()
	var v int64
	if err := e.db.QueryRow(`SELECT amount FROM payments WHERE transaction_id = ?`, id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// stored is the Currency as the application loaded it, with the id the test
// stored it under; a case's Currency has no id of its own.
func (e *matrixEnv) stored(c moneytest.Currency) Currency { return e.curs[c.Code] }

func TestAmountsStoredSummedExactlyEveryCurrency(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		e := newMatrixEnv(t, moneytest.Currencies[0])
		moneytest.ForEachAmount(t, moneytest.Valid(c.Decimals), func(t *testing.T, a moneytest.Amount) {
			if a.Minor == 0 {
				return // a payment of 0 is refused on new input
			}
			before := e.balance(t, c)
			in, err := e.tx(t, "income", c, a.Input)
			if err != nil {
				t.Fatal(err)
			}
			if got := e.rawAmount(t, in.ID); got != a.Minor {
				t.Fatalf("stored %d, want %d", got, a.Minor)
			}
			if got, line := in.Payments[0].Amount, in.Lines[0].Amount; got.Minor() != a.Minor ||
				got.Unit() != e.stored(c).Unit() || line.Minor() != a.Minor {
				t.Fatalf("read back payment %s (%+v), line %s", got, got.Unit(), line)
			}
			if got := e.balance(t, c).Sub(before); got.Minor() != a.Minor {
				t.Fatalf("balance moved by %s, want %d minor units", got, a.Minor)
			}
			if _, err := e.tx(t, "expense", c, a.Input); err != nil {
				t.Fatal(err)
			}
			if got := e.balance(t, c); !got.Equal(before) {
				t.Fatalf("income then expense left %s, want %s", got, before)
			}
		})
	})
}

func TestManyMaximumAmountsSumExactly(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		e := newMatrixEnv(t, moneytest.Currencies[0])
		max := decimal.New(money.MaxMinor, -int32(c.Decimals))
		for i := 0; i < 50; i++ {
			if _, err := e.tx(t, "income", c, max); err != nil {
				t.Fatal(err)
			}
		}
		if got := e.balance(t, c); got.Minor() != 50*money.MaxMinor {
			t.Fatalf("balance = %d, want %d", got.Minor(), 50*money.MaxMinor)
		}
	})
}

func TestAmountsOutRangeRejectedStoreNothing(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		e := newMatrixEnv(t, moneytest.Currencies[0])
		acct := e.accts[c.Code]
		cur := e.stored(c)
		moneytest.ForEachAmount(t, moneytest.Invalid(c.Decimals), func(t *testing.T, a moneytest.Amount) {
			for _, in := range []decimal.Decimal{a.Input, a.Input.Neg()} {
				txs, accts, budgets := e.count(t, "transactions"), e.count(t, "accounts"), e.count(t, "budgets")
				date := "2024-01-15"
				check := func(what string, err error) {
					t.Helper()
					if !errors.Is(err, money.ErrOutOfRange) {
						t.Errorf("%s with %s: err = %v, want ErrOutOfRange", what, in, err)
					}
				}
				x := simple("income", acct.ID, "1")
				x.Date, x.Payments[0].Amount = &date, &in
				_, err := e.txs.Create(e.ctx, x)
				check("payment", err)
				x = simple("income", acct.ID, "1")
				x.Date, x.Lines[0].Amount = &date, in
				_, err = e.txs.Create(e.ctx, x)
				check("line", err)
				_, err = (Accounts{DB: e.db}).Create(
					e.ctx,
					AccountInput{Name: "big", Type: "cash", CurrencyID: cur.ID, InitialBalance: in, IsActive: true},
				)
				check("account balance", err)
				_, err = (Debts{Accounts: Accounts{DB: e.db}, Transactions: e.txs}).Create(
					e.ctx,
					DebtInput{Name: "debt", Direction: DebtIOwe, CurrencyID: cur.ID, Amount: in},
				)
				check("debt principal", err)
				_, err = (Budgets{DB: e.db}).Create(
					e.ctx,
					BudgetInput{Name: "b", Amount: in, CurrencyID: &cur.ID, Period: "monthly"},
				)
				check("budget", err)

				if got := e.count(t, "transactions"); got != txs {
					t.Errorf("%d transactions stored, want %d", got, txs)
				}
				if got := e.count(t, "accounts"); got != accts {
					t.Errorf("%d accounts stored, want %d", got, accts)
				}
				if got := e.count(t, "budgets"); got != budgets {
					t.Errorf("%d budgets stored, want %d", got, budgets)
				}
				if e.count(t, "payments") != 0 || e.count(t, "transaction_lines") != 0 {
					t.Error("a payment or line of a rejected transaction was stored")
				}
			}
		})
	})
}

func TestUpdateRejectsRangeKeepsAmount(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		e := newMatrixEnv(t, moneytest.Currencies[0])
		orig, err := e.tx(t, "income", c, decimal.New(12345, -int32(c.Decimals)))
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range moneytest.Invalid(c.Decimals) {
			date := "2024-01-15"
			x := simple("income", orig.Payments[0].AccountID, "1")
			x.Date, x.Payments[0].Amount = &date, &a.Input
			_, err := e.txs.Update(e.ctx, orig.ID, x)
			if !errors.Is(err, money.ErrOutOfRange) {
				t.Errorf("%s: err = %v", a.Name, err)
			}
		}
		if got := e.rawAmount(t, orig.ID); got != 12345 {
			t.Errorf("amount changed to %d", got)
		}
	})
}

func TestConvertEveryPairAmount(t *testing.T) {
	// Rates against USD, so the base of this matrix is USD.
	stored := map[string]Currency{}
	for i, c := range moneytest.Currencies {
		stored[c.Code] = Currency{ID: int64(i + 1), Code: c.Code, Decimals: c.Decimals, Rate: c.Rate, IsBase: c.Code == "USD"}
	}
	moneytest.ForEachPair(t, func(t *testing.T, from, to moneytest.Currency) {
		f, tt := stored[from.Code], stored[to.Code]
		moneytest.ForEachAmount(t, moneytest.Valid(from.Decimals), func(t *testing.T, a moneytest.Amount) {
			in := money.New(a.Minor, f.Unit())
			exact := moneytest.Converted(in.Decimal(), from, to)
			want, fits := money.FromInput(exact, tt.Unit())

			got, err := TryConvert(in, f, tt)
			if fits != nil {
				// Beyond MaxMinor: refused for a user's amount; Convert, which
				// serves computed values, only gives up beyond int64.
				if !errors.Is(err, money.ErrOutOfRange) {
					t.Fatalf("err = %v, want ErrOutOfRange for %s", err, exact)
				}
				scaled := exact.Round(int32(tt.Decimals)).Shift(int32(tt.Decimals))
				if scaled.BigInt().IsInt64() {
					if c := Convert(in, f, tt); c.Minor() != scaled.IntPart() {
						t.Fatalf("Convert = %s, want %s", c, exact)
					}
				} else {
					mustPanic(t, func() { Convert(in, f, tt) })
				}
				return
			}
			if err != nil || !got.Equal(want) {
				t.Fatalf("TryConvert = %s, %v; want %s", got, err, want)
			}
			if c := Convert(in, f, tt); !c.Equal(want) {
				t.Fatalf("Convert = %s, want %s", c, want)
			}
			if got.Unit() != tt.Unit() {
				t.Fatalf("result unit %+v, want %+v", got.Unit(), tt.Unit())
			}
		})
	})
}

func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic")
		}
	}()
	fn()
}

// A there-and-back conversion may lose at most what rounding to the two scales
// can take, never more.
func TestConvertRoundTripErrorBoundedScales(t *testing.T) {
	stored := map[string]Currency{}
	for i, c := range moneytest.Currencies {
		stored[c.Code] = Currency{ID: int64(i + 1), Code: c.Code, Decimals: c.Decimals, Rate: c.Rate, IsBase: c.Code == "USD"}
	}
	half := decimal.New(5, -1)
	moneytest.ForEachPair(t, func(t *testing.T, from, to moneytest.Currency) {
		f, tt := stored[from.Code], stored[to.Code]
		for _, a := range moneytest.Valid(from.Decimals) {
			in := money.New(a.Minor, f.Unit())
			there, err := TryConvert(in, f, tt)
			if err != nil {
				continue
			}
			back, err := TryConvert(there, tt, f)
			if err != nil {
				continue
			}
			// Rounding into `to` moves the value by up to half a unit of `to`,
			// which is worth rate_to/rate_from units of `from` on the way back.
			unitFrom := decimal.New(1, -int32(from.Decimals))
			unitTo := decimal.New(1, -int32(to.Decimals))
			bound := half.Mul(unitFrom).
				Add(half.Mul(unitTo).Mul(to.Rate).DivRound(from.Rate, 40)).
				Add(unitFrom.Mul(half))
				// + final rounding
			diff := back.Decimal().Sub(in.Decimal()).Abs()
			if diff.GreaterThan(bound) {
				t.Errorf("%s: %s -> %s -> %s drifted by %s, bound %s", a.Name, in, there, back, diff, bound)
			}
		}
	})
}

// The amount of each side of a transfer is rounded into the scale of its own
// account, whatever the currency on the other side.
func TestTransferPaymentsRoundToTheirAccounts(t *testing.T) {
	moneytest.ForEach(t, func(t *testing.T, to moneytest.Currency) {
		e := newMatrixEnv(t, moneytest.Currencies[0])
		from := moneytest.Currencies[0]
		fromAcct, toAcct := e.accts[from.Code], e.accts[to.Code]
		if from.Code == to.Code {
			t.Skip("a transfer needs two accounts")
		}
		moneytest.ForEachAmount(t, moneytest.Valid(to.Decimals), func(t *testing.T, a moneytest.Amount) {
			if a.Minor == 0 {
				return
			}
			date := "2024-01-15"
			x := move(fromAcct.ID, toAcct.ID, "1", a.Input.String())
			x.Date = &date
			tx, err := e.txs.Create(e.ctx, x)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = e.txs.Delete(e.ctx, tx.ID) }()
			for _, p := range tx.Payments {
				if p.AccountID == toAcct.ID && (p.Amount.Minor() != a.Minor || p.Amount.Unit() != e.stored(to).Unit()) {
					t.Fatalf("destination payment = %v, want %d minor", p.Amount, a.Minor)
				}
			}
		})
	})
}
