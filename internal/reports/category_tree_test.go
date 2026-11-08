package reports

import (
	"testing"

	"savvy-go/internal/ledger"
)

// subEnv adds Leisure (no lines of its own unless a test adds them) with
// Cinema and Games under it, beside the top-level Food and Fun.
type subEnv struct {
	*seriesEnv
	leisure, cinema, games ledger.Category
}

func newSubEnv(t *testing.T) *subEnv {
	t.Helper()
	e := &subEnv{seriesEnv: newSeriesEnv(t)}
	cats := ledger.Categories{DB: e.db}
	mk := func(name string, parent *int64) ledger.Category {
		c, err := cats.Create(e.ctx, ledger.Category{Name: name, Type: "expense", Color: ptrTo("#123456"), ParentID: parent})
		if err != nil {
			t.Fatal(err)
		}
		return *c
	}
	e.leisure = mk("Leisure", nil)
	e.cinema = mk("Cinema", &e.leisure.ID)
	e.games = mk("Games", &e.leisure.ID)
	return e
}

// The dashboard summary shows top-level categories only: a parent carries its
// children's lines, even when it has none of its own.
func TestCategorySummaryRollsChildrenIntoParent(t *testing.T) {
	e := newSubEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.cinema.ID, "40", "2024-01-14")
	e.add(t, "expense", e.jpyAcc.ID, &e.games.ID, "1000", "2024-01-14") // 10 USD
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "30", "2024-01-14")

	cats, total := (Service{DB: e.db}).CategorySummary(e.ctx, janRange, "expense")
	if len(cats) != 2 || cats[0].Name != "Leisure" || cats[1].Name != "Food" {
		t.Fatalf("want Leisure then Food, got %+v", cats)
	}
	wantMoney(t, "leisure", *cats[0].TotalAmount, usd(e.moneyEnv, "50"))
	if cats[0].Color == nil || *cats[0].Color != "#123456" || cats[0].ParentID != nil {
		t.Errorf("parent keeps its own look and stays top level: %+v", cats[0])
	}
	wantMoney(t, "total", total, usd(e.moneyEnv, "80"))
}

func TestCategorySummaryParentOwnLinesJoinChildren(t *testing.T) {
	e := newSubEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.leisure.ID, "5", "2024-01-14")
	e.add(t, "expense", e.usdAcc.ID, &e.cinema.ID, "40", "2024-01-14")

	cats, _ := (Service{DB: e.db}).CategorySummary(e.ctx, janRange, "expense")
	if len(cats) != 1 {
		t.Fatalf("one top-level category, got %+v", cats)
	}
	wantMoney(t, "leisure", *cats[0].TotalAmount, usd(e.moneyEnv, "45"))
}

// The overview list gives each parent its group's total and lists the parts:
// its children and, when it has any, its own lines under its own id.
func TestExpensesCategoryNestsChildren(t *testing.T) {
	e := newSubEnv(t)
	e.add(t, "expense", e.usdAcc.ID, &e.cinema.ID, "40", "2024-02-10")
	e.add(t, "expense", e.usdAcc.ID, &e.leisure.ID, "5", "2024-02-10")
	e.add(t, "expense", e.usdAcc.ID, &e.games.ID, "7", "2024-01-10")   // before only
	e.add(t, "expense", e.usdAcc.ID, &e.cinema.ID, "20", "2024-01-11") // before
	e.add(t, "expense", e.usdAcc.ID, &e.food.ID, "30", "2024-02-11")
	f := Filter{PeriodType: "month", PeriodValue: "2024-02", CompareWith: "previous_period"}

	got := (Service{DB: e.db}).ExpensesByCategory(e.ctx, f)
	if len(got.Categories) != 2 || got.Categories[0].Name != "Leisure" || got.Categories[1].Name != "Food" {
		t.Fatalf("want Leisure then Food, got %+v", got.Categories)
	}
	l := got.Categories[0]
	wantMoney(t, "leisure now", l.Current, usd(e.moneyEnv, "45"))
	wantMoney(t, "leisure before", l.Previous, usd(e.moneyEnv, "27"))
	if len(l.Children) != 2 || l.Children[0].Name != "Cinema" || l.Children[1].ID != e.leisure.ID {
		t.Fatalf("children %+v", l.Children)
	}
	wantMoney(t, "cinema before", l.Children[0].Previous, usd(e.moneyEnv, "20"))
	if l.Children[0].ParentID != e.leisure.ID {
		t.Errorf("child points at its parent: %+v", l.Children[0])
	}
	if len(got.Categories[1].Children) != 0 {
		t.Errorf("a category without subcategories lists no parts: %+v", got.Categories[1])
	}
}
