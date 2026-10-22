package refsync

import (
	"testing"

	"savvy-go/internal/ledger"
	"savvy-go/internal/store"
)

// drift is what makes the spaces with the indexes differ.
func (e *env) drift(t *testing.T, idx ...int) Diff {
	t.Helper()
	var ids []int64
	for _, i := range idx {
		ids = append(ids, e.id(i))
	}
	var d Diff
	err := e.st.InSpaces(ctx, ids, func(dbs map[int64]store.DB) error {
		var err error
		d, err = diffSpaces(ctx, dbs, ids, []Kind{Categories, Tags})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// A crash between the commits of two spaces, or a restore, leaves the group
// apart. At startup it is brought together by adding, never by deleting.
func TestHealAddsWhatMissingNeverDeletes(t *testing.T) {
	e := newEnv(t)
	e.group(t, 0, 1, 2)
	// Drift, made behind the group's back: B has a category and a tag the
	// others lack, C looks different and has another default, A lost nothing.
	if _,
		err := e.cats(t, 1).Create(ctx, ledger.Category{Name: "Pets", Type: "expense", Color: ptr("green")}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.tags(t, 1).Create(ctx, "vet"); err != nil {
		t.Fatal(err)
	}
	food := e.category(t, 0, "expense", "Food")
	if _,
		err := e.cats(t, 0).Update(ctx, food.ID,
		ledger.Category{Name: "Food", Type: "expense", Color: ptr("red"), IsDefault: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.cats(t, 2).Create(ctx, ledger.Category{Name: "Gym", Type: "expense"}); err != nil {
		t.Fatal(err)
	}
	gym := e.category(t, 2, "expense", "Gym")
	if _, err := e.cats(t, 2).SetDefault(ctx, gym.ID); err != nil {
		t.Fatal(err)
	}
	if e.drift(t, 0, 1, 2).Empty() {
		t.Fatal("the fixture did not drift")
	}

	if err := e.svc.Heal(ctx); err != nil {
		t.Fatal(err)
	}
	if d := e.drift(t, 0, 1, 2); !d.Empty() {
		t.Fatalf("still apart: %+v", d)
	}
	for i := 0; i < 3; i++ {
		names := e.names(t, i, "expense")
		if !has(names, "Pets") || !has(names, "Gym") || !has(names, "Food") {
			t.Fatalf("space %d lost or lacks a category: %v", i, names)
		}
		if list, _ := e.tags(t, i).All(ctx); len(list) != 1 || list[0].Name != "vet" {
			t.Fatalf("space %d: tags = %+v", i, list)
		}
		// The lowest space (A) wins: its look and its default.
		if c := e.category(t, i, "expense", "Food"); c.Color == nil || *c.Color != "red" || !c.IsDefault {
			t.Fatalf("space %d: Food = %+v", i, c)
		}
	}
	// Nothing left to do the second time.
	if err := e.svc.Heal(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestHealLeavesIdenticalUngroupedSpacesAlone(t *testing.T) {
	e := newEnv(t)
	e.group(t, 0, 1)
	e.seed(t, 2) // not in the group
	if _, err := e.cats(t, 2).Create(ctx, ledger.Category{Name: "Solo", Type: "expense"}); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Heal(ctx); err != nil {
		t.Fatal(err)
	}
	if e.category(t, 0, "expense", "Solo") != nil || e.category(t, 1, "expense", "Solo") != nil {
		t.Fatal("a space outside the group leaked into it")
	}
}

func TestHealSkipsKindIsSwitchedOff(t *testing.T) {
	e := newEnv(t)
	e.seed(t, 0)
	e.seed(t, 1)
	ids := []int64{e.id(0), e.id(1)}
	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: Change{Add: ids, Categories: true}}); err != nil {
		t.Fatal(err)
	}
	_, _ = e.tags(t, 0).Create(ctx, "solo")
	if err := e.svc.Heal(ctx); err != nil {
		t.Fatal(err)
	}
	if list, _ := e.tags(t, 1).All(ctx); len(list) != 0 {
		t.Fatalf("tags were healed with the flag off: %+v", list)
	}
}
