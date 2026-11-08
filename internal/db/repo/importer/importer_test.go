package importer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/db/filter"
	importerrepo "savvy-go/internal/db/repo/importer"
	ledgerrepo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
)

var ctx = context.Background()

func TestImportLifecycle(t *testing.T) {
	r := importerrepo.Repo{DB: dbtest.Space(t)}
	now := time.Now().UTC()
	if im, err := r.ByID(ctx, "nope"); im != nil || err != nil {
		t.Fatalf("unknown import = %v, %v", im, err)
	}
	// user_id points at the server's users; a space database only keeps the number.
	if err := r.Insert(ctx, "i1", 7, "up1", "parsing", now); err != nil {
		t.Fatal(err)
	}
	im, _ := r.ByID(ctx, "i1")
	if im == nil || im.Status != "parsing" || im.UserID == nil || *im.UserID != 7 || im.UploadID == nil ||
		im.TotalRows != nil || im.Mapping != nil || im.Meta != nil || im.Message != nil {
		t.Fatalf("new import = %+v", im)
	}

	_ = r.MarkParsed(ctx, "i1", 120, `{"headers":["a"]}`, now)
	im, _ = r.ByID(ctx, "i1")
	if im.Status != "parsed" || im.TotalRows == nil || *im.TotalRows != 120 || im.Meta == nil ||
		*im.Meta != `{"headers":["a"]}` {
		t.Fatalf("parsed = %+v", im)
	}

	_ = r.MarkImporting(ctx, "i1", `{"date":0}`, `{"x":1}`, now)
	im, _ = r.ByID(ctx, "i1")
	if im.Status != "importing" || im.Mapping == nil || *im.Mapping != `{"date":0}` || im.Options == nil {
		t.Fatalf("importing = %+v", im)
	}

	_ = r.MarkCompleted(
		ctx,
		"i1",
		importerrepo.Result{Processed: 120, Created: 100, Skipped: 15, Errors: 5, ErrorsJSON: `[]`, MetaJSON: `{}`},
		now,
	)
	im, _ = r.ByID(ctx, "i1")
	if im.Status != "completed" || im.ProcessedRows != 120 || im.CreatedCount != 100 || im.SkippedCount != 15 ||
		im.ErrorCount != 5 {
		t.Fatalf("completed = %+v", im)
	}

	_ = r.Insert(ctx, "i2", 7, "up2", "parsing", now)
	_ = r.Fail(ctx, "i2", "The file is gone.", now)
	if im, _ := r.ByID(ctx, "i2"); im.Status != "failed" || im.Message == nil || *im.Message != "The file is gone." {
		t.Fatalf("failed = %+v", im)
	}

	counts, err := r.CountsByStatus(ctx)
	if err != nil || counts["completed"] != 1 || counts["failed"] != 1 || len(counts) != 2 {
		t.Fatalf("counts = %v, %v", counts, err)
	}
}

func TestRowsWithSameHashInsertedOnce(t *testing.T) {
	d := dbtest.Space(t)
	l := ledgerrepo.Repo{DB: d}
	now := time.Now().UTC()
	usd, _ := l.InsertCurrency(
		ctx,
		ledgerrepo.Currency{Code: "USD", Name: "Dollar", Symbol: "$", Decimals: 2, IsBase: true, Rate: decimal.NewFromInt(1)},
		now,
	)
	acc, _ := l.InsertAccount(
		ctx,
		ledgerrepo.AccountFields{
			Name:           "Cash",
			Type:           "cash",
			CurrencyID:     usd,
			InitialBalance: money.New(0, money.Unit{ID: usd, Decimals: 2}),
			IsActive:       true,
		},
		now,
	)

	r := importerrepo.Repo{DB: d}
	row := importerrepo.Row{
		Type:        "expense",
		AccountID:   acc,
		Minor:       1250,
		Description: "Coffee",
		Date:        "2026-02-01",
		Status:      "confirmed",
		DedupHash:   "h1",
	}
	if added, err := r.InsertRow(ctx, row, now); !added || err != nil {
		t.Fatalf("first insert = %v, %v", added, err)
	}
	if added, err := r.InsertRow(ctx, row, now); added || err != nil {
		t.Fatalf("a duplicate must be skipped, not fail: %v, %v", added, err)
	}
	row.DedupHash = "h2"
	if added, _ := r.InsertRow(ctx, row, now); !added {
		t.Fatal("a different hash is a different row")
	}
	if n := l.CountTransactions(ctx); n != 2 {
		t.Fatalf("%d transactions", n)
	}
}

func TestImportedRowIsOnePaymentAndOneLine(t *testing.T) {
	d := dbtest.Space(t)
	l := ledgerrepo.Repo{DB: d}
	now := time.Now().UTC()
	jpy, _ := l.InsertCurrency(
		ctx,
		ledgerrepo.Currency{Code: "JPY", Name: "Yen", Symbol: "Y", Decimals: 0, IsBase: true, Rate: decimal.NewFromInt(1)},
		now,
	)
	acc, _ := l.InsertAccount(
		ctx,
		ledgerrepo.AccountFields{Name: "Yen", Type: "cash", CurrencyID: jpy, IsActive: true},
		now,
	)
	cat, _ := l.InsertCategory(ctx, ledgerrepo.Category{Name: "Food", Type: "expense"}, now)

	r := importerrepo.Repo{DB: d}
	for _, row := range []importerrepo.Row{
		{Type: "expense", AccountID: acc, CategoryID: &cat, Minor: 800, Date: "2026-02-01", Status: "confirmed", DedupHash: "a"},
		{Type: "income", AccountID: acc, Minor: 5000, Date: "2026-02-02", Status: "confirmed", DedupHash: "b"},
	} {
		if added, err := r.InsertRow(ctx, row, now); !added || err != nil {
			t.Fatalf("insert = %v, %v", added, err)
		}
	}
	// The same hash on another account is another row.
	other, _ := l.InsertAccount(ctx, ledgerrepo.AccountFields{Name: "Other", Type: "cash", CurrencyID: jpy, IsActive: true}, now)
	row := importerrepo.Row{Type: "expense", AccountID: other, Minor: 1, Status: "confirmed", DedupHash: "a"}
	if added, err := r.InsertRow(ctx, row, now); !added || err != nil {
		t.Fatalf("same hash, other account = %v, %v", added, err)
	}

	changes, _ := l.DailyChanges(ctx, acc, "")
	if len(changes) != 2 || changes[0].Delta != -800 || changes[1].Delta != 5000 {
		t.Fatalf("changes = %+v", changes)
	}
	if has, _ := l.HasExternalID(ctx, "csv", fmt.Sprintf("%d:a", acc)); !has {
		t.Fatal("the external source is csv and the id <account>:<hash>")
	}
	if rows, _ := l.SummaryRows(ctx, filter.TxFilter{}, "confirmed"); len(rows) != 3 {
		t.Fatalf("every imported row has one line: %+v", rows)
	}
}

// A batch stores in one transaction what InsertRow stores one by one: it skips
// duplicates, also those inside the batch, and a row that fails leaves nothing
// behind and does not stop the others.
func TestInsertRowsKeepsGoodRowsAroundBadOne(t *testing.T) {
	d := dbtest.Space(t)
	l := ledgerrepo.Repo{DB: d}
	now := time.Now().UTC()
	usd, _ := l.InsertCurrency(
		ctx,
		ledgerrepo.Currency{Code: "USD", Name: "Dollar", Symbol: "$", Decimals: 2, IsBase: true, Rate: decimal.NewFromInt(1)},
		now,
	)
	acc, _ := l.InsertAccount(
		ctx,
		ledgerrepo.AccountFields{
			Name: "Cash", Type: "cash", CurrencyID: usd,
			InitialBalance: money.New(0, money.Unit{ID: usd, Decimals: 2}), IsActive: true,
		},
		now,
	)
	r := importerrepo.Repo{DB: d}
	row := func(hash string, account int64) importerrepo.Row {
		return importerrepo.Row{
			Type: "expense", AccountID: account, Minor: 100, Description: "x", Date: "2026-02-01",
			Status: "confirmed", DedupHash: hash,
		}
	}
	got, err := r.InsertRows(ctx, []importerrepo.Row{row("a", acc), row("a", acc), row("b", 9999), row("c", acc)}, now)
	if err != nil || len(got) != 4 {
		t.Fatalf("InsertRows = %v, %v", got, err)
	}
	want := []string{"added", "skipped", "failed", "added"}
	for i, res := range got {
		state := "skipped"
		switch {
		case res.Err != nil:
			state = "failed"
		case res.Added:
			state = "added"
		}
		if state != want[i] {
			t.Errorf("row %d is %s, want %s", i+1, state, want[i])
		}
	}
	if n := l.CountTransactions(ctx); n != 2 {
		t.Fatalf("%d transactions, want 2 (the failed row must leave nothing)", n)
	}
}
