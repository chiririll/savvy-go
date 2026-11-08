package ledger

import "testing"

func TestCategoryParent(t *testing.T) {
	e := newMoneyEnv(t)
	cats := Categories{DB: e.db}
	mk := func(name, typ string, parent *int64) *Category {
		t.Helper()
		c, err := cats.Create(e.ctx, Category{Name: name, Type: typ, ParentID: parent})
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		return c
	}
	food := mk("Food", "expense", nil)
	cafe := mk("Cafe", "expense", &food.ID)
	if cafe.ParentID == nil || *cafe.ParentID != food.ID {
		t.Fatalf("parent = %v, want %d", cafe.ParentID, food.ID)
	}

	bad := map[string]Category{
		"other type":     {Name: "Bonus", Type: "income", ParentID: &food.ID},
		"unknown parent": {Name: "Ghost", Type: "expense", ParentID: ptr(int64(9999))},
		"grandchild":     {Name: "Latte", Type: "expense", ParentID: &cafe.ID},
	}
	for name, c := range bad {
		if _, err := cats.Create(e.ctx, c); err == nil {
			t.Errorf("%s: created", name)
		}
	}

	if _, err := cats.Update(e.ctx, food.ID, Category{Name: "Food", Type: "expense", ParentID: &food.ID}); err == nil {
		t.Error("a category became its own parent")
	}
	other := mk("Home", "expense", nil)
	if _, err := cats.Update(e.ctx, food.ID, Category{Name: "Food", Type: "expense", ParentID: &other.ID}); err == nil {
		t.Error("a category with children got a parent")
	}
	got, err := cats.Update(e.ctx, cafe.ID, Category{Name: "Cafe", Type: "expense", ParentID: &other.ID})
	if err != nil || *got.ParentID != other.ID {
		t.Fatalf("move: %v %+v", err, got)
	}
	got, err = cats.Update(e.ctx, cafe.ID, Category{Name: "Cafe", Type: "expense"})
	if err != nil || got.ParentID != nil {
		t.Fatalf("detach: %v %+v", err, got)
	}
}

func TestDeletingParentFreesChildren(t *testing.T) {
	e := newMoneyEnv(t)
	cats := Categories{DB: e.db}
	// A second category of the type keeps the delete from being the last one.
	if _, err := cats.Create(e.ctx, Category{Name: "Spare", Type: "expense"}); err != nil {
		t.Fatal(err)
	}
	food, _ := cats.Create(e.ctx, Category{Name: "Food", Type: "expense"})
	cafe, _ := cats.Create(e.ctx, Category{Name: "Cafe", Type: "expense", ParentID: &food.ID})
	if err := cats.Delete(e.ctx, food.ID, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := cats.ByID(e.ctx, cafe.ID)
	if got == nil || got.ParentID != nil {
		t.Fatalf("child after parent's deletion = %+v, want top level", got)
	}
}
