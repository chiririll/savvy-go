package importer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/dbtest"
	"savvy-go/internal/db/filter"
	authrepo "savvy-go/internal/db/repo/auth"
	"savvy-go/internal/ledger"
	"savvy-go/internal/store"
	"savvy-go/internal/uploads"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

type env struct {
	ctx     context.Context
	svc     Service
	space   store.DB
	userID  int64
	usd     ledger.Currency
	btc     ledger.Currency
	usdAcct ledger.Account
	btcAcct ledger.Account
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	server, space := dbtest.Server(t), dbtest.Space(t)
	uid, err := authrepo.Repo{DB: server}.InsertUser(ctx, "Ann", "ann@test.com", "h", "user", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	e := &env{
		ctx: ctx, space: space, userID: uid,
		svc: Service{
			DB:      space,
			Uploads: uploads.Service{DB: server, Root: t.TempDir(), SignSecret: "s"},
			Txs:     ledger.Transactions{DB: space},
		},
	}
	curs := ledger.Currencies{DB: space}
	mk := func(code string, decimals int, rate string) ledger.Currency {
		c, err := curs.Create(ctx, ledger.Currency{Code: code, Name: code, Symbol: code, Decimals: decimals, Rate: dec(rate)})
		if err != nil {
			t.Fatal(err)
		}
		return *c
	}
	e.usd, e.btc = mk("USD", 2, "1"), mk("BTC", 8, "50000")
	accts := ledger.Accounts{DB: space}
	acct := func(name string, c ledger.Currency) ledger.Account {
		a, err := accts.Create(ctx, ledger.AccountInput{Name: name, Type: "cash", CurrencyID: c.ID, IsActive: true})
		if err != nil {
			t.Fatal(err)
		}
		return *a
	}
	e.usdAcct, e.btcAcct = acct("usd", e.usd), acct("btc", e.btc)
	return e
}

// upload stores csv as a finished upload and returns its id.
func (e *env) upload(t *testing.T, csv string) string {
	t.Helper()
	up := e.svc.Uploads
	size := int64(len(csv))
	u, err := up.Create(e.ctx, e.userID, "transaction-imports", "statement.csv", &size, nil)
	if err != nil {
		t.Fatal(err)
	}
	etag, _, err := up.StorePart(u, 1, strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	done, err := up.Complete(e.ctx, u, []struct {
		Number int
		ETag   string
	}{{1, etag}})
	if err != nil {
		t.Fatal(err)
	}
	return done.ID
}

// parsed uploads csv and runs the parse step.
func (e *env) parsed(t *testing.T, csv string) *Import {
	t.Helper()
	uploadID := e.upload(t, csv)
	im, err := e.svc.Create(e.ctx, e.userID, uploadID)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Parse(e.ctx, im.ID, uploadID); err != nil {
		t.Fatal(err)
	}
	im, err = e.svc.ByID(e.ctx, im.ID)
	if err != nil || im == nil {
		t.Fatal(err)
	}
	return im
}

func (e *env) run(t *testing.T, im *Import, accountID int64, options map[string]any) *Import {
	t.Helper()
	if options == nil {
		options = map[string]any{}
	}
	options["default_account_id"] = float64(accountID)
	if _, ok := options["date_format"]; !ok {
		options["date_format"], options["amount_format"] = "ISO", "US"
	}
	mapping := map[string]any{}
	for k, v := range im.Meta.SuggestedMapping {
		mapping[k] = float64(v)
	}
	if err := e.svc.Execute(e.ctx, im.ID, mapping, options); err != nil {
		t.Fatal(err)
	}
	done, _ := e.svc.ByID(e.ctx, im.ID)
	return done
}

func (e *env) transactions(t *testing.T) []ledger.Transaction {
	t.Helper()
	list, _, err := e.svc.Txs.Filtered(e.ctx, filter.TxFilter{}, 1, 500)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

func TestParseReadsHeadersFormatsAndAPreview(t *testing.T) {
	e := newEnv(t)
	var b strings.Builder
	b.WriteString("Дата;Сумма;Описание\n")
	for i := 1; i <= 12; i++ {
		b.WriteString("0" + string(rune('0'+i%9+1)) + ".01.2024;-1,50;Coffee\n")
	}
	im := e.parsed(t, b.String())

	if im.Status != "parsed" {
		t.Fatalf("status %q after parsing", im.Status)
	}
	if im.TotalRows == nil || *im.TotalRows != 12 {
		t.Errorf("total rows %v", im.TotalRows)
	}
	if len(im.Meta.PreviewRows) != 10 {
		t.Errorf("the preview is the first ten rows: %d", len(im.Meta.PreviewRows))
	}
	m := im.Meta
	if m.ParseMeta.Delimiter != ";" || m.DetectedFormats.DateFormat != "DD.MM.YYYY" ||
		m.DetectedFormats.AmountFormat != "EU" {
		t.Errorf("detected %+v", m)
	}
	if m.SuggestedMapping["date"] != 0 || m.SuggestedMapping["amount"] != 1 || m.SuggestedMapping["description"] != 2 {
		t.Errorf("mapping %v", m.SuggestedMapping)
	}
}

// Excel puts a byte order mark in front of the first header.
func TestCSVWithByteOrderMarkMapsItsFirstColumn(t *testing.T) {
	e := newEnv(t)
	im := e.parsed(t, "\xef\xbb\xbfDate,Amount,Description\n2024-01-01,-5.00,Tea\n")
	if len(im.Meta.Headers) == 0 || im.Meta.Headers[0] != "Date" {
		t.Fatalf("headers %q", im.Meta.Headers)
	}
	if i, ok := im.Meta.SuggestedMapping["date"]; !ok || i != 0 {
		t.Errorf("the date column was not found: %v", im.Meta.SuggestedMapping)
	}
}

func TestAFileWithOnlyHeadersHasNothingToImport(t *testing.T) {
	e := newEnv(t)
	im := e.parsed(t, "Date,Amount,Description\n")
	if im.Status != "parsed" || im.TotalRows == nil || *im.TotalRows != 0 {
		t.Fatalf("import %+v", im)
	}
	done := e.run(t, im, e.usdAcct.ID, nil)
	if done.Status != "completed" || done.CreatedCount != 0 || done.ErrorCount != 0 {
		t.Errorf("import %+v", done)
	}
}

func TestExecuteCreatesConfirmedTransactionsInTheAccountCurrency(t *testing.T) {
	e := newEnv(t)
	im := e.parsed(t, "Date,Amount,Description\n2024-01-01,-12.50,Coffee\n2024-01-02,2000,Salary\n2024-01-03,-0.99,Gum\n")
	done := e.run(t, im, e.usdAcct.ID, nil)

	if done.Status != "completed" || done.CreatedCount != 3 || done.SkippedCount != 0 || done.ErrorCount != 0 ||
		done.ProcessedRows != 3 {
		t.Fatalf("import %+v", done)
	}
	byDesc := map[string]ledger.Transaction{}
	for _, tx := range e.transactions(t) {
		byDesc[*tx.Description] = tx
	}
	coffee, salary := byDesc["Coffee"], byDesc["Salary"]
	if coffee.Type != "expense" || coffee.Amount.Decimal().String() != "12.5" || coffee.Status != "confirmed" ||
		coffee.AccountID != e.usdAcct.ID {
		t.Errorf("coffee %+v", coffee)
	}
	if salary.Type != "income" || salary.Amount.Decimal().String() != "2000" {
		t.Errorf("salary %+v", salary)
	}
	if byDesc["Gum"].Amount.Minor() != 99 {
		t.Errorf("gum is 99 cents: %d", byDesc["Gum"].Amount.Minor())
	}
}

func TestReimportingTheSameFileCreatesNothingNew(t *testing.T) {
	e := newEnv(t)
	csv := "Date,Amount,Description\n2024-01-01,-12.50,Coffee\n2024-01-02,-3.00,  coffee  \n"
	first := e.run(t, e.parsed(t, csv), e.usdAcct.ID, nil)
	second := e.run(t, e.parsed(t, csv), e.usdAcct.ID, nil)
	if first.CreatedCount != 2 || second.CreatedCount != 0 || second.SkippedCount != 2 {
		t.Errorf("first %+v second %+v", first, second)
	}
	if n := len(e.transactions(t)); n != 2 {
		t.Errorf("%d transactions", n)
	}
	// The same rows in another account are not duplicates of these.
	other := e.run(t, e.parsed(t, csv), e.btcAcct.ID, nil)
	if other.CreatedCount != 2 {
		t.Errorf("another account: %+v", other)
	}
}

// Two small amounts of a currency with many decimals must not look like one.
func TestDifferentAmountsBelowACentAreNotDuplicates(t *testing.T) {
	e := newEnv(t)
	csv := "Date,Amount,Description\n2024-01-01,0.00000150,Mining payout\n2024-01-01,0.00000250,Mining payout\n2024-01-01,1.001,Dust\n2024-01-01,1.004,Dust\n"
	done := e.run(t, e.parsed(t, csv), e.btcAcct.ID, nil)
	if done.CreatedCount != 4 || done.SkippedCount != 0 {
		t.Fatalf("rows with different amounts were taken for duplicates: %+v", done)
	}
	var total decimal.Decimal
	for _, tx := range e.transactions(t) {
		total = total.Add(tx.Amount.Decimal())
	}
	if !total.Equal(dec("2.005004")) {
		t.Errorf("total %s", total)
	}
}

func TestDedupHashIgnoresCaseAndSpacingButNotAmounts(t *testing.T) {
	base := dedupHash("2024-01-01", dec("12.5"), "Coffee Shop")
	if dedupHash("2024-01-01", dec("12.50"), "  coffee   SHOP ") != base {
		t.Error("the same row with other spacing and case is the same row")
	}
	for name, h := range map[string]string{
		"another date":   dedupHash("2024-01-02", dec("12.5"), "Coffee Shop"),
		"another amount": dedupHash("2024-01-01", dec("12.51"), "Coffee Shop"),
		"a tiny change":  dedupHash("2024-01-01", dec("12.500001"), "Coffee Shop"),
		"another text":   dedupHash("2024-01-01", dec("12.5"), "Tea Shop"),
	} {
		if h == base {
			t.Errorf("%s hashes alike", name)
		}
	}
}

func TestExecuteReportsRowsItCannotTake(t *testing.T) {
	e := newEnv(t)
	csv := "Date,Amount,Description\n2024-01-01,-5,ok\nnot a date,-5,bad date\n2024-01-02,lots,bad amount\n2999-01-01,-5,future\n2024-01-04\n"
	done := e.run(t, e.parsed(t, csv), e.usdAcct.ID, nil)

	if done.CreatedCount != 1 || done.ErrorCount != 4 || done.ProcessedRows != 5 {
		t.Fatalf("counts %+v", done)
	}
	rows := map[int]string{}
	for _, re := range done.Errors {
		rows[re.Row] = re.Message
	}
	for row, want := range map[int]string{2: "Invalid date", 3: "Invalid amount", 4: "future", 5: "too short"} {
		if !strings.Contains(rows[row], want) {
			t.Errorf("row %d: %q, want %q", row, rows[row], want)
		}
	}
}

func TestExecuteOptionsDefaultTypeAndExplicitTypeColumn(t *testing.T) {
	e := newEnv(t)
	csv := "Date,Amount,Description,Type\n2024-01-01,10,A,income\n2024-01-02,10,B,Expense\n2024-01-03,10,C,\n"
	done := e.run(t, e.parsed(t, csv), e.usdAcct.ID, map[string]any{"default_type": "expense"})
	if done.CreatedCount != 3 {
		t.Fatalf("%+v", done)
	}
	types := map[string]string{}
	for _, tx := range e.transactions(t) {
		types[*tx.Description] = tx.Type
	}
	if types["A"] != "income" || types["B"] != "expense" {
		t.Errorf("an explicit type column wins: %v", types)
	}
	if types["C"] != "income" {
		t.Errorf("a blank type cell falls back to the sign of the amount, so a positive amount is income: %q", types["C"])
	}
}

func TestExecuteResolvesCategoriesByTheUsersChoices(t *testing.T) {
	e := newEnv(t)
	cats := ledger.Categories{DB: e.space}
	food, _ := cats.Create(e.ctx, ledger.Category{Name: "Food", Type: "expense"})
	fun, _ := cats.Create(e.ctx, ledger.Category{Name: "Fun", Type: "expense"})
	csv := "Date,Amount,Description,Category\n" +
		"2024-01-01,-1,a,food\n" + // matches Food by name
		"2024-01-02,-2,b,Rent\n" + // new
		"2024-01-03,-3,c,Gym\n" + // sent to Fun
		"2024-01-04,-4,d,Misc\n" + // skipped
		"2024-01-05,-5,e,\n"
	done := e.run(t, e.parsed(t, csv), e.usdAcct.ID, map[string]any{
		"create_missing_categories": true,
		"category_map":              map[string]any{"gym": float64(fun.ID), "misc": "skip"},
	})
	if done.CreatedCount != 5 {
		t.Fatalf("%+v", done)
	}
	got := map[string]*int64{}
	for _, tx := range e.transactions(t) {
		got[*tx.Description] = tx.CategoryID
	}
	if got["a"] == nil || *got["a"] != food.ID {
		t.Errorf("food: %v", got["a"])
	}
	if got["c"] == nil || *got["c"] != fun.ID {
		t.Errorf("gym: %v", got["c"])
	}
	if got["d"] != nil || got["e"] != nil {
		t.Errorf("skipped and blank categories stay empty: %v %v", got["d"], got["e"])
	}
	if got["b"] == nil {
		t.Fatal("Rent should have been created")
	}
	if len(done.Meta.CreatedCategories) != 1 || done.Meta.CreatedCategories[0] != "Rent" {
		t.Errorf("created categories %v", done.Meta.CreatedCategories)
	}
	// The same file with creation off leaves unknown names uncategorised.
	again := e.run(t, e.parsed(t, "Date,Amount,Description,Category\n2024-02-01,-9,z,Brandnew\n"), e.usdAcct.ID, nil)
	if again.CreatedCount != 1 || len(again.Meta.CreatedCategories) != 0 {
		t.Errorf("%+v", again)
	}
}

func TestPreviewSaysWhatExecuteWouldDo(t *testing.T) {
	e := newEnv(t)
	cats := ledger.Categories{DB: e.space}
	if _, err := cats.Create(e.ctx, ledger.Category{Name: "Food", Type: "expense"}); err != nil {
		t.Fatal(err)
	}
	im := e.parsed(t, "Date,Amount,Description,Category\n2024-01-01,-1,a,Food\n2024-01-02,-2,b,Rent\nbad,-3,c,Rent\n")
	mapping := map[string]any{}
	for k, v := range im.Meta.SuggestedMapping {
		mapping[k] = float64(v)
	}
	res, err := e.svc.Preview(e.ctx, im, mapping, map[string]any{"date_format": "ISO", "amount_format": "US"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.WillCreate != 2 || res.Summary.HasErrors != 1 || len(res.Transactions) != 3 {
		t.Fatalf("summary %+v", res.Summary)
	}
	if res.Transactions[2].Status != "error" || res.Transactions[2].Error == nil {
		t.Errorf("row 3 %+v", res.Transactions[2])
	}
	if len(res.Summary.CategoriesToCreate) != 1 || res.Summary.CategoriesToCreate[0] != "Rent" {
		t.Errorf("to create %v", res.Summary.CategoriesToCreate)
	}
	if n := len(e.transactions(t)); n != 0 {
		t.Errorf("a preview stores nothing: %d transactions", n)
	}
	// A preview does not consume the file.
	if _, err := e.svc.Preview(e.ctx, im, mapping, nil); err != nil {
		t.Errorf("second preview: %v", err)
	}
}

func TestPreviewAndExecuteNeedTheFile(t *testing.T) {
	e := newEnv(t)
	im := e.parsed(t, "Date,Amount\n2024-01-01,-1\n")
	done := e.run(t, im, e.usdAcct.ID, nil)
	if done.Status != "completed" {
		t.Fatalf("%+v", done)
	}
	// The file is discarded once imported.
	if _, err := e.svc.Preview(e.ctx, done, map[string]any{"date": float64(0), "amount": float64(1)}, nil); err == nil {
		t.Error("previewing an import whose file is gone")
	}
	if err := e.svc.Execute(e.ctx, done.ID, map[string]any{}, map[string]any{}); err != nil {
		t.Fatal(err)
	}
	got, _ := e.svc.ByID(e.ctx, done.ID)
	if got.Status != "failed" {
		t.Errorf("importing again without the file: status %q", got.Status)
	}
	if err := e.svc.Execute(e.ctx, "nope", nil, nil); err != nil {
		t.Errorf("an unknown import is ignored: %v", err)
	}
}

func TestCountsByStatus(t *testing.T) {
	e := newEnv(t)
	e.parsed(t, "Date,Amount\n2024-01-01,-1\n")
	e.run(t, e.parsed(t, "Date,Amount\n2024-01-02,-1\n"), e.usdAcct.ID, nil)
	counts, err := e.svc.CountsByStatus(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, n := range counts {
		total += n
	}
	if total != 2 || counts["completed"] != 1 {
		t.Errorf("counts %v", counts)
	}
}

func TestParseImportTypeAndMappingHelpers(t *testing.T) {
	for in, want := range map[string]string{"income": "income", "Income": "income", " expense ": "expense", "": "", "transfer": ""} {
		if got := parseImportType(in); got != want {
			t.Errorf("parseImportType(%q) = %q, want %q", in, got, want)
		}
	}
	if i, ok := mappingIndex(map[string]any{"a": float64(2), "b": "3", "c": "x"}, "a"); !ok || i != 2 {
		t.Error("mappingIndex float")
	}
	if i, ok := mappingIndex(map[string]any{"b": "3"}, "b"); !ok || i != 3 {
		t.Error("mappingIndex string")
	}
	if _, ok := mappingIndex(map[string]any{"c": "x"}, "c"); ok {
		t.Error("mappingIndex of a word")
	}
	if _, ok := mappingIndex(nil, "a"); ok {
		t.Error("mappingIndex of nothing")
	}
	if nilOr("") != nil || *nilOr("x") != "x" {
		t.Error("nilOr")
	}
}
