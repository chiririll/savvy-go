package reports

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/ledger"
	"savvy-go/internal/migrate"
	"savvy-go/internal/money/moneytest"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func openSpaceDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := migrate.Space.Up(context.Background(), sqlDB); err != nil {
		t.Fatal(err)
	}
	return sqlDB
}

// moneyEnv is a space with USD (the base), JPY and BTC, one account each.
type moneyEnv struct {
	ctx            context.Context
	db             *sql.DB
	usd, jpy, btc  ledger.Currency
	usdAcc, jpyAcc ledger.Account
	btcAcc         ledger.Account
	txs            ledger.Transactions
}

func newMoneyEnv(t *testing.T) *moneyEnv {
	t.Helper()
	sqlDB := openSpaceDB(t)
	ctx := context.Background()
	env := &moneyEnv{ctx: ctx, db: sqlDB, txs: ledger.Transactions{DB: sqlDB}}
	mk := func(code string, decimals int, rate string) ledger.Currency {
		c, err := (ledger.Currencies{DB: sqlDB}).Create(ctx, ledger.Currency{Code: code, Name: code, Symbol: code, Decimals: decimals, Rate: dec(rate)})
		if err != nil {
			t.Fatal(err)
		}
		return *c
	}
	env.usd = mk("USD", 2, "1")
	env.jpy = mk("JPY", 0, "0.01")
	env.btc = mk("BTC", 8, "50000")
	acct := func(name string, cur ledger.Currency, initial string) ledger.Account {
		a, err := (ledger.Accounts{DB: sqlDB}).Create(ctx, ledger.AccountInput{Name: name, Type: "cash", CurrencyID: cur.ID, InitialBalance: dec(initial), IsActive: true})
		if err != nil {
			t.Fatal(err)
		}
		return *a
	}
	env.usdAcc = acct("usd", env.usd, "100")
	env.jpyAcc = acct("jpy", env.jpy, "3000")
	env.btcAcc = acct("btc", env.btc, "0")
	return env
}

func (e *moneyEnv) tx(t *testing.T, typ string, accountID int64, amount string) *ledger.Transaction {
	t.Helper()
	date := "2024-01-15"
	tx, err := e.txs.Create(e.ctx, ledger.TxInput{Type: typ, AccountID: accountID, Amount: dec(amount), Date: &date})
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

// matrixEnv is a space holding every moneytest currency, one account each,
// with base as the base currency.
type matrixEnv struct {
	ctx   context.Context
	db    *sql.DB
	curs  map[string]ledger.Currency
	accts map[string]ledger.Account
	txs   ledger.Transactions
}

func newMatrixEnv(t *testing.T, base moneytest.Currency) *matrixEnv {
	t.Helper()
	sqlDB := openSpaceDB(t)
	ctx := context.Background()
	e := &matrixEnv{ctx: ctx, db: sqlDB, curs: map[string]ledger.Currency{}, accts: map[string]ledger.Account{}, txs: ledger.Transactions{DB: sqlDB}}
	add := func(c moneytest.Currency) {
		created, err := (ledger.Currencies{DB: sqlDB}).Create(ctx, ledger.Currency{
			Code: c.Code, Name: c.Code, Symbol: c.Code, Decimals: c.Decimals, Rate: c.RateIn(base),
		})
		if err != nil {
			t.Fatalf("create %v: %v", c, err)
		}
		e.curs[c.Code] = *created
		acct, err := (ledger.Accounts{DB: sqlDB}).Create(ctx, ledger.AccountInput{Name: c.Code, Type: "cash", CurrencyID: created.ID, IsActive: true})
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
func (e *matrixEnv) tx(t *testing.T, typ string, c moneytest.Currency, amount decimal.Decimal) (*ledger.Transaction, error) {
	t.Helper()
	date := "2024-01-15"
	return e.txs.Create(e.ctx, ledger.TxInput{Type: typ, AccountID: e.accts[c.Code].ID, Amount: amount, Date: &date})
}

// stored is the Currency as the application loaded it, with the id the test
// stored it under; a case's Currency has no id of its own.
func (e *matrixEnv) stored(c moneytest.Currency) ledger.Currency { return e.curs[c.Code] }
