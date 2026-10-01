package domain

import (
	"context"
	"path/filepath"
	"testing"

	"savvy-go/internal/db"
	"savvy-go/internal/migrate"
)

func TestDetectDateFormat(t *testing.T) {
	cases := []struct {
		name   string
		values []string
		want   string
	}{
		{"iso", []string{"2024-01-05", "2024-02-17", "2024-03-01 10:20:30"}, "ISO"},
		{"dots", []string{"05.01.2024", "17.02.2024", "1.3.2024"}, "DD.MM.YYYY"},
		{"day first slashes", []string{"05/01/2024", "17/02/2024", "01/03/2024"}, "DD/MM/YYYY"},
		{"month first slashes", []string{"01/05/2024", "02/17/2024", "03/01/2024"}, "MM/DD/YYYY"},
		{"ambiguous prefers day first", []string{"01/02/2024", "03/04/2024"}, "DD/MM/YYYY"},
	}
	for _, c := range cases {
		if got := detectDateFormat(c.values); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestDetectAmountFormat(t *testing.T) {
	cases := []struct {
		name      string
		values    []string
		delimiter rune
		want      string
	}{
		{"us", []string{"1,234.56", "-12.50", "100"}, ',', "US"},
		{"eu", []string{"1.234,56", "-12,50", "100"}, ';', "EU"},
		{"eu decimals only", []string{"12,5", "3,99", "100,00"}, ',', "EU"},
		{"us thousands only", []string{"1,234,567", "2,000"}, ',', "US"},
		{"eu thousands only", []string{"1.234.567", "2.000"}, ';', "EU"},
		{"ambiguous semicolon file", []string{"1,234", "2,500"}, ';', "EU"},
		{"ambiguous comma file", []string{"1,234", "2,500"}, ',', "US"},
		{"currency symbols", []string{"$1,234.56", "€ 12.00"}, ',', "US"},
	}
	for _, c := range cases {
		if got := detectAmountFormat(c.values, c.delimiter); got != c.want {
			t.Errorf("%s: got %s want %s", c.name, got, c.want)
		}
	}
}

func TestParseImportAmount(t *testing.T) {
	cases := []struct {
		in, format string
		want       float64
	}{
		{"1,234.56", "US", 1234.56},
		{"1.234,56", "EU", 1234.56},
		{"-12,50", "EU", -12.5},
		{"(12.50)", "US", -12.5},
		{"12.50-", "US", -12.5},
		{"1 234,5 ₽", "EU", 1234.5},
	}
	for _, c := range cases {
		got, err := parseImportAmount(c.in, c.format)
		if err != nil || got != c.want {
			t.Errorf("%q (%s): got %v, %v want %v", c.in, c.format, got, err, c.want)
		}
	}
}

func TestDetectImportFindsColumnsWithoutHeaders(t *testing.T) {
	headers := []string{"col1", "col2", "col3"}
	var rows [][]string
	for i := 1; i <= 20; i++ {
		rows = append(rows, []string{"Shop", dayString(i), "12,50"})
	}
	d := detectImport(headers, rows, ';')
	if d.mapping["date"] != 1 || d.mapping["amount"] != 2 {
		t.Fatalf("mapping %v", d.mapping)
	}
	if d.dateFormat != "DD.MM.YYYY" || d.amountFormat != "EU" {
		t.Fatalf("formats %s %s", d.dateFormat, d.amountFormat)
	}
}

func dayString(i int) string {
	return string(rune('0'+i/10)) + string(rune('0'+i%10)) + ".03.2024"
}

func TestSniffDelimiter(t *testing.T) {
	if d := sniffDelimiter([]byte("\xef\xbb\xbfdate;amount;memo\n")); d != ';' {
		t.Fatalf("got %q", d)
	}
	if d := sniffDelimiter([]byte("date,amount\n")); d != ',' {
		t.Fatalf("got %q", d)
	}
}

func TestCategoryResolver(t *testing.T) {
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	ctx := context.Background()
	if err := migrate.Up(ctx, sqlDB); err != nil {
		t.Fatal(err)
	}
	cats := Categories{DB: sqlDB}
	food, err := cats.Create(ctx, Category{Name: "Food", Type: "expense"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := cats.Create(ctx, Category{Name: "Groceries", Type: "expense"})
	if err != nil {
		t.Fatal(err)
	}

	rows := [][]string{
		{"2024-01-01", "-5", "food"},
		{"2024-01-02", "-6", "Food"},
		{"2024-01-03", "-7", "Cafes"},
		{"2024-01-04", "-8", "Supermarket"},
		{"2024-01-05", "-9", "Gym"},
		{"2024-01-06", "100", "Salary"},
		{"2024-01-07", "-1", ""},
	}
	mapping := map[string]any{"date": float64(0), "amount": float64(1), "category": float64(2), "type": nil}
	options := map[string]any{"date_format": "ISO", "amount_format": "US", "default_type": "expense"}

	found := collectImportCategories(rows, mapping, options)
	if len(found) != 5 || found[0].Name != "food" || found[0].Count != 2 {
		t.Fatalf("collected %+v", found)
	}
	imports := Imports{DB: sqlDB}
	if err := imports.matchImportCategories(ctx, found); err != nil {
		t.Fatal(err)
	}
	if found[0].MatchID == nil || *found[0].MatchID != food.ID {
		t.Fatalf("food should match the existing category: %+v", found[0])
	}
	for _, c := range found[1:] {
		if c.MatchID != nil {
			t.Fatalf("%s should be unmatched", c.Name)
		}
		if c.Name == "Salary" && c.Type != "income" {
			t.Fatalf("salary type %s", c.Type)
		}
	}

	options["category_map"] = map[string]any{
		"Supermarket": float64(other.ID), // map to an existing category
		"Cafes":       "create",          // create a new one
		"Gym":         "skip",            // leave uncategorized
	}
	options["create_missing_categories"] = false
	r, err := imports.newCategoryResolver(ctx, found, options)
	if err != nil {
		t.Fatal(err)
	}
	if got := r.resolve("food"); !got.Valid || got.Int64 != food.ID {
		t.Fatalf("food: %+v", got)
	}
	if got := r.resolve("Supermarket"); !got.Valid || got.Int64 != other.ID {
		t.Fatalf("supermarket: %+v", got)
	}
	if got := r.resolve("Gym"); got.Valid {
		t.Fatalf("gym should be skipped: %+v", got)
	}
	if got := r.resolve("Salary"); got.Valid {
		t.Fatalf("salary has no choice and create_missing is off: %+v", got)
	}
	cafes := r.resolve("Cafes")
	if !cafes.Valid || r.resolve("cafes") != cafes {
		t.Fatalf("cafes: %+v", cafes)
	}
	if len(r.created) != 1 || r.created[0] != "Cafes" {
		t.Fatalf("created %v", r.created)
	}

	options["create_missing_categories"] = true
	r, _ = imports.newCategoryResolver(ctx, found, options)
	salary := r.resolve("Salary")
	if !salary.Valid {
		t.Fatal("salary should be created when create_missing is on")
	}
	created, _ := cats.ByID(ctx, salary.Int64)
	if created == nil || created.Type != "income" {
		t.Fatalf("created salary %+v", created)
	}
}
