package refsync

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/ledger"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/store/sqlite"
)

var ctx = context.Background()

type env struct {
	st    *sqlite.Store
	svc   Service
	owner *auth.User
	spc   []*spaces.Space // A, B, C
}

func newEnv(t *testing.T) *env {
	t.Helper()
	st, err := sqlite.OpenApp(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sps := spaces.Service{Store: st}
	e := &env{st: st, svc: Service{Spaces: sps}, owner: e0(t, st, "Owner", "o@test.com")}
	for _, name := range []string{"A", "B", "C"} {
		sp, err := sps.Create(ctx, name, e.owner)
		if err != nil {
			t.Fatal(err)
		}
		e.spc = append(e.spc, sp)
	}
	return e
}

func e0(t *testing.T, st store.Store, name, email string) *auth.User {
	t.Helper()
	pass := "secret1"
	u, err := (auth.Users{DB: st.Server()}).Create(ctx, name, email, &pass, auth.RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func (e *env) id(i int) int64 { return e.spc[i].ID }

func (e *env) db(t *testing.T, i int) store.DB {
	t.Helper()
	d, err := e.st.Space(ctx, e.id(i))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func (e *env) cats(t *testing.T, i int) ledger.Categories { return ledger.Categories{DB: e.db(t, i)} }
func (e *env) tags(t *testing.T, i int) ledger.Tags       { return ledger.Tags{DB: e.db(t, i)} }

// seed gives space i the categories every test starts from: a default
// expense and a default income category.
func (e *env) seed(t *testing.T, i int) {
	t.Helper()
	for _, c := range []ledger.Category{
		{Name: "Food", Type: "expense", IsDefault: true},
		{Name: "Salary", Type: "income", IsDefault: true},
	} {
		if _, err := e.cats(t, i).Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
}

// group puts the spaces with the indexes in one group that syncs both kinds,
// the spaces being identical (seeded) already.
func (e *env) group(t *testing.T, idx ...int) {
	t.Helper()
	var ids []int64
	for _, i := range idx {
		e.seed(t, i)
		ids = append(ids, e.id(i))
	}
	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: Change{Add: ids, Categories: true, Tags: true}}); err != nil {
		t.Fatal(err)
	}
}

func (e *env) names(t *testing.T, i int, typ string) []string {
	t.Helper()
	list, err := e.cats(t, i).All(ctx, typ)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, c := range list {
		out = append(out, c.Name)
	}
	return out
}

func (e *env) category(t *testing.T, i int, typ, name string) *ledger.Category {
	t.Helper()
	c, err := findCategory(ctx, e.cats(t, i), typ, name)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// spend books an expense of the category in space i.
func (e *env) spend(t *testing.T, i int, categoryID int64) {
	t.Helper()
	d := e.db(t, i)
	cur, err := (ledger.Currencies{DB: d}).Create(ctx, ledger.Currency{Code: "EUR", Name: "EUR", Symbol: "EUR", Decimals: 2, IsBase: true, Rate: decimal.NewFromInt(1)})
	if err != nil {
		t.Fatal(err)
	}
	acc, err := (ledger.Accounts{DB: d}).Create(ctx, ledger.AccountInput{Name: "Cash", Type: "cash", CurrencyID: cur.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.ExecContext(ctx, `INSERT INTO transactions (type, account_id, category_id, amount, status, date) VALUES ('expense', ?, ?, 100, 'confirmed', '2026-01-01')`, acc.ID, categoryID); err != nil {
		t.Fatal(err)
	}
}

func (e *env) count(t *testing.T, i int, categoryID int64) int {
	t.Helper()
	c, err := e.cats(t, i).ByID(ctx, categoryID)
	if err != nil || c == nil {
		t.Fatalf("category %d: %v", categoryID, err)
	}
	return c.TransactionsCount
}

func has(list []string, want string) bool {
	for _, s := range list {
		if strings.EqualFold(s, want) {
			return true
		}
	}
	return false
}

func TestOperationsAreMirroredToTheWholeGroup(t *testing.T) {
	e := newEnv(t)
	e.group(t, 0, 1, 2)
	a := e.id(0)

	created, err := e.svc.CreateCategory(ctx, e.owner, a, ledger.Category{Name: "Rent", Type: "expense", Color: ptr("#112233")})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if c := e.category(t, i, "expense", "rent"); c == nil || c.Color == nil || *c.Color != "#112233" {
			t.Fatalf("space %d: Rent = %+v", i, c)
		}
	}

	// A rename and an edit follow by the old name.
	if _, err := e.svc.UpdateCategory(ctx, e.owner, a, created.ID, ledger.Category{Name: "Housing", Type: "expense", Color: ptr("#445566")}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if names := e.names(t, i, "expense"); has(names, "Rent") || !has(names, "Housing") {
			t.Fatalf("space %d: names = %v", i, names)
		}
		if c := e.category(t, i, "expense", "Housing"); *c.Color != "#445566" {
			t.Fatalf("space %d: color = %v", i, *c.Color)
		}
	}

	// The default follows too.
	if _, err := e.svc.SetDefaultCategory(ctx, e.owner, a, created.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if !e.category(t, i, "expense", "Housing").IsDefault || e.category(t, i, "expense", "Food").IsDefault {
			t.Fatalf("space %d: the default did not move", i)
		}
	}

	// Tags: create, rename, delete.
	tag, err := e.svc.CreateTag(ctx, e.owner, a, "trip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.RenameTag(ctx, e.owner, a, tag.ID, "holiday"); err != nil {
		t.Fatal(err)
	}
	for i := 1; i < 3; i++ {
		list, _ := e.tags(t, i).All(ctx)
		if len(list) != 1 || list[0].Name != "holiday" {
			t.Fatalf("space %d: tags = %+v", i, list)
		}
	}
	if err := e.svc.DeleteTag(ctx, e.owner, a, tag.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if list, _ := e.tags(t, i).All(ctx); len(list) != 0 {
			t.Fatalf("space %d: tags after delete = %+v", i, list)
		}
	}
}

func TestDeletingMovesTransactionsOfEverySpaceToOneSuccessor(t *testing.T) {
	e := newEnv(t)
	e.group(t, 0, 1, 2)
	for _, name := range []string{"Taxi", "Transport"} {
		if _, err := e.svc.CreateCategory(ctx, e.owner, e.id(0), ledger.Category{Name: name, Type: "expense"}); err != nil {
			t.Fatal(err)
		}
	}
	e.spend(t, 0, e.category(t, 0, "expense", "Taxi").ID) // A and B use Taxi, C does not
	e.spend(t, 1, e.category(t, 1, "expense", "Taxi").ID)

	taxi := e.category(t, 0, "expense", "Taxi")
	// No successor although a space uses the category.
	if err := e.svc.DeleteCategory(ctx, e.owner, e.id(0), taxi.ID, nil); err == nil {
		t.Fatal("deleted a category with transactions and no successor")
	}
	if e.category(t, 2, "expense", "Taxi") == nil {
		t.Fatal("a failed delete must roll every space back")
	}

	// A and B use it, C does not: the one successor chosen in A serves all three.
	succ := e.category(t, 0, "expense", "Transport")
	if err := e.svc.DeleteCategory(ctx, e.owner, e.id(0), taxi.ID, &succ.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if e.category(t, i, "expense", "Taxi") != nil {
			t.Fatalf("space %d still has Taxi", i)
		}
	}
	for _, i := range []int{0, 1} {
		if n := e.count(t, i, e.category(t, i, "expense", "Transport").ID); n != 1 {
			t.Fatalf("space %d: Transport has %d transactions, want 1", i, n)
		}
	}
	if n := e.count(t, 2, e.category(t, 2, "expense", "Transport").ID); n != 0 {
		t.Fatalf("space C: Transport has %d transactions, want 0", n)
	}
}

func TestADeleteWithTransactionsOnlyInAnotherSpaceNeedsTheSuccessor(t *testing.T) {
	e := newEnv(t)
	e.group(t, 0, 1)
	for _, name := range []string{"Taxi", "Transport"} {
		if _, err := e.svc.CreateCategory(ctx, e.owner, e.id(0), ledger.Category{Name: name, Type: "expense"}); err != nil {
			t.Fatal(err)
		}
	}
	e.spend(t, 1, e.category(t, 1, "expense", "Taxi").ID) // only B

	taxi := e.category(t, 0, "expense", "Taxi")
	err := e.svc.DeleteCategory(ctx, e.owner, e.id(0), taxi.ID, nil)
	var m *MirrorError
	if !errors.As(err, &m) || m.Space != e.id(1) {
		t.Fatalf("err = %v, want a mirror error for space B", err)
	}
	// A had nothing to move and was deleted from first; B's refusal undoes it.
	if e.category(t, 0, "expense", "Taxi") == nil {
		t.Fatal("the delete in A was not rolled back")
	}
}

func TestEveryAffectedSpaceNeedsAnEditor(t *testing.T) {
	e := newEnv(t)
	e.group(t, 0, 1)
	// A second user edits A but only views B.
	bob := e0(t, e.st, "Bob", "bob@test.com")
	if err := e.svc.Spaces.SetMember(ctx, e.id(0), bob, spaces.RoleEditor); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Spaces.SetMember(ctx, e.id(1), bob, spaces.RoleViewer); err != nil {
		t.Fatal(err)
	}
	_, err := e.svc.CreateCategory(ctx, bob, e.id(0), ledger.Category{Name: "Gifts", Type: "expense"})
	if !errors.Is(err, ErrRights) {
		t.Fatalf("err = %v, want ErrRights", err)
	}
	if e.category(t, 0, "expense", "Gifts") != nil {
		t.Fatal("the denied change was applied")
	}
}

func TestKindsAreSwitchedIndependently(t *testing.T) {
	e := newEnv(t)
	e.seed(t, 0)
	e.seed(t, 1)
	ids := []int64{e.id(0), e.id(1)}
	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: Change{Add: ids, Categories: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.CreateTag(ctx, e.owner, e.id(0), "solo"); err != nil {
		t.Fatal(err)
	}
	if list, _ := e.tags(t, 1).All(ctx); len(list) != 0 {
		t.Fatalf("tags were mirrored with the flag off: %+v", list)
	}
	if _, err := e.svc.CreateCategory(ctx, e.owner, e.id(0), ledger.Category{Name: "Pets", Type: "expense"}); err != nil {
		t.Fatal(err)
	}
	if e.category(t, 1, "expense", "Pets") == nil {
		t.Fatal("categories were not mirrored")
	}
}

func TestWithoutAGroupNothingIsMirrored(t *testing.T) {
	e := newEnv(t)
	e.seed(t, 0)
	e.seed(t, 1)
	if _, err := e.svc.CreateCategory(ctx, e.owner, e.id(0), ledger.Category{Name: "Pets", Type: "expense"}); err != nil {
		t.Fatal(err)
	}
	if e.category(t, 1, "expense", "Pets") != nil {
		t.Fatal("a lone space changed another")
	}
}

func TestEnablingNeedsEveryDifferenceDecided(t *testing.T) {
	e := newEnv(t)
	e.seed(t, 0)
	e.seed(t, 1)
	// A has Rent (red) and Gym; B has Rent (blue) and Books; their defaults agree.
	for _, c := range []ledger.Category{{Name: "Rent", Type: "expense", Color: ptr("red")}, {Name: "Gym", Type: "expense"}} {
		if _, err := e.cats(t, 0).Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []ledger.Category{{Name: "Rent", Type: "expense", Color: ptr("blue")}, {Name: "Books", Type: "expense"}} {
		if _, err := e.cats(t, 1).Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	ids := []int64{e.id(0), e.id(1)}
	change := Change{Add: ids, Categories: true}

	diff, err := e.svc.Diff(ctx, e.owner, change)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Categories) != 3 || len(diff.Defaults) != 0 || len(diff.Tags) != 0 {
		t.Fatalf("diff = %+v", diff)
	}

	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: change}); !errors.Is(err, ErrUnresolved) {
		t.Fatalf("applying without decisions: %v", err)
	}
	if g, _ := e.svc.GroupOf(ctx, e.id(0)); g != nil {
		t.Fatal("a refused plan created the group")
	}

	plan := Plan{Change: change, Decisions: []Decision{
		{Kind: Categories, Type: "expense", Name: "Rent", Keep: true, From: e.id(1)},
		{Kind: Categories, Type: "expense", Name: "Gym", Keep: true},
		{Kind: Categories, Type: "expense", Name: "Books", Keep: false},
	}}
	if _, err := e.svc.Apply(ctx, e.owner, plan); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		names := e.names(t, i, "expense")
		if !has(names, "Gym") || !has(names, "Rent") || has(names, "Books") || len(names) != 3 {
			t.Fatalf("space %d: %v", i, names)
		}
		if c := e.category(t, i, "expense", "Rent"); *c.Color != "blue" {
			t.Fatalf("space %d: Rent color = %v, want blue", i, *c.Color)
		}
	}
	if d := e.drift(t, 0, 1); !d.Empty() {
		t.Fatalf("the spaces differ after enabling: %+v", d)
	}
}

func TestDifferingDefaultsNeedADecision(t *testing.T) {
	e := newEnv(t)
	e.seed(t, 0)
	e.seed(t, 1)
	for i := 0; i < 2; i++ {
		if _, err := e.cats(t, i).Create(ctx, ledger.Category{Name: "Rent", Type: "expense"}); err != nil {
			t.Fatal(err)
		}
	}
	rent := e.category(t, 1, "expense", "Rent")
	if _, err := e.cats(t, 1).SetDefault(ctx, rent.ID); err != nil { // A: Food, B: Rent
		t.Fatal(err)
	}
	change := Change{Add: []int64{e.id(0), e.id(1)}, Categories: true}
	diff, _ := e.svc.Diff(ctx, e.owner, change)
	if len(diff.Defaults) != 1 || diff.Defaults[0].Type != "expense" {
		t.Fatalf("diff = %+v", diff)
	}
	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: change}); !errors.Is(err, ErrUnresolved) {
		t.Fatalf("err = %v", err)
	}
	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: change, Defaults: []DefaultDecision{{Type: "expense", Name: "Rent"}}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if !e.category(t, i, "expense", "Rent").IsDefault {
			t.Fatalf("space %d: Rent is not the default", i)
		}
	}
}

func TestDroppingACategoryDuringEnablingMovesItsTransactions(t *testing.T) {
	e := newEnv(t)
	e.seed(t, 0)
	e.seed(t, 1)
	if _, err := e.cats(t, 1).Create(ctx, ledger.Category{Name: "Books", Type: "expense"}); err != nil {
		t.Fatal(err)
	}
	e.spend(t, 1, e.category(t, 1, "expense", "Books").ID)
	change := Change{Add: []int64{e.id(0), e.id(1)}, Categories: true}

	drop := Decision{Kind: Categories, Type: "expense", Name: "Books", Keep: false}
	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: change, Decisions: []Decision{drop}}); err == nil {
		t.Fatal("dropped a category that has transactions without a successor")
	}
	drop.Successor = "Food"
	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: change, Decisions: []Decision{drop}}); err != nil {
		t.Fatal(err)
	}
	if n := e.count(t, 1, e.category(t, 1, "expense", "Food").ID); n != 1 {
		t.Fatalf("Food has %d transactions, want the 1 moved from Books", n)
	}
}

func TestTagsAreReconciledToo(t *testing.T) {
	e := newEnv(t)
	e.seed(t, 0)
	e.seed(t, 1)
	_, _ = e.tags(t, 0).Create(ctx, "keepme")
	_, _ = e.tags(t, 1).Create(ctx, "dropme")
	change := Change{Add: []int64{e.id(0), e.id(1)}, Tags: true}
	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: change}); !errors.Is(err, ErrUnresolved) {
		t.Fatalf("err = %v", err)
	}
	plan := Plan{Change: change, Decisions: []Decision{
		{Kind: Tags, Name: "keepme", Keep: true}, {Kind: Tags, Name: "dropme", Keep: false},
	}}
	if _, err := e.svc.Apply(ctx, e.owner, plan); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if list, _ := e.tags(t, i).All(ctx); len(list) != 1 || list[0].Name != "keepme" {
			t.Fatalf("space %d: tags = %+v", i, list)
		}
	}
}

func TestMembershipRules(t *testing.T) {
	e := newEnv(t)
	e.seed(t, 0)
	e.seed(t, 1)
	e.seed(t, 2)
	ab := []int64{e.id(0), e.id(1)}

	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: Change{Add: ab[:1], Categories: true}}); err == nil {
		t.Fatal("a group of one space was created")
	}
	g, err := e.svc.Apply(ctx, e.owner, Plan{Change: Change{Add: ab, Categories: true}})
	if err != nil {
		t.Fatal(err)
	}
	// A space is in one group only.
	if _, err := e.svc.Apply(ctx, e.owner, Plan{Change: Change{Add: []int64{e.id(1), e.id(2)}, Categories: true}}); err == nil {
		t.Fatal("a space joined a second group")
	}

	// A third space joins and is reconciled with the whole group.
	g, err = e.svc.Apply(ctx, e.owner, Plan{Change: Change{GroupID: g.ID, Add: []int64{e.id(2)}, Categories: true}})
	if err != nil || len(g.Members) != 3 {
		t.Fatalf("group = %+v, %v", g, err)
	}

	// Only an admin of the space may take it out; removing needs no one else.
	bob := e0(t, e.st, "Bob", "bob@test.com")
	_ = e.svc.Spaces.SetMember(ctx, e.id(2), bob, spaces.RoleAdmin)
	if _, err := e.svc.Apply(ctx, bob, Plan{Change: Change{GroupID: g.ID, Remove: []int64{e.id(0)}, Categories: true}}); !errors.Is(err, ErrNotAdmin) {
		t.Fatalf("err = %v, want ErrNotAdmin", err)
	}
	if g, err = e.svc.Apply(ctx, bob, Plan{Change: Change{GroupID: g.ID, Remove: []int64{e.id(2)}, Categories: true}}); err != nil || len(g.Members) != 2 {
		t.Fatalf("group = %+v, %v", g, err)
	}

	// A group that falls below two spaces is dissolved.
	if g, err = e.svc.Apply(ctx, e.owner, Plan{Change: Change{GroupID: g.ID, Remove: []int64{e.id(0)}, Categories: true}}); err != nil || g != nil {
		t.Fatalf("group = %+v, %v", g, err)
	}
	if g, _ := e.svc.GroupOf(ctx, e.id(1)); g != nil {
		t.Fatalf("group of the last space = %+v", g)
	}
}

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

func ptr(s string) *string { return &s }

// A crash between the commits of two spaces, or a restore, leaves the group
// apart. At startup it is brought together by adding, never by deleting.
func TestHealAddsWhatIsMissingAndNeverDeletes(t *testing.T) {
	e := newEnv(t)
	e.group(t, 0, 1, 2)
	// Drift, made behind the group's back: B has a category and a tag the
	// others lack, C looks different and has another default, A lost nothing.
	if _, err := e.cats(t, 1).Create(ctx, ledger.Category{Name: "Pets", Type: "expense", Color: ptr("green")}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.tags(t, 1).Create(ctx, "vet"); err != nil {
		t.Fatal(err)
	}
	food := e.category(t, 0, "expense", "Food")
	if _, err := e.cats(t, 0).Update(ctx, food.ID, ledger.Category{Name: "Food", Type: "expense", Color: ptr("red"), IsDefault: true}); err != nil {
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

func TestHealLeavesIdenticalAndUngroupedSpacesAlone(t *testing.T) {
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

func TestHealSkipsAKindThatIsSwitchedOff(t *testing.T) {
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
