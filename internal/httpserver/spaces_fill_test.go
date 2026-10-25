//go:build integration

package httpserver

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/ledger"
	"savvy-go/internal/spaces"
	"savvy-go/internal/spaces/spacefill"
)

// fixedRates is a rate source with exactly the currencies and rates it is given
// (a zero rate means "unknown"), and no live rates.
type fixedRates map[string]string

func (f fixedRates) Catalog(_ context.Context, _ string) []ledger.Currency {
	meta := map[string]ledger.Currency{
		"USD": {Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2},
		"EUR": {Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2},
		"GBP": {Code: "GBP", Name: "Pound", Symbol: "£", Decimals: 2},
		"JPY": {Code: "JPY", Name: "Yen", Symbol: "¥", Decimals: 0},
	}
	var out []ledger.Currency
	for code, c := range meta {
		c.Rate = decimal.RequireFromString(f.rate(code))
		out = append(out, c)
	}
	return out
}

func (f fixedRates) rate(code string) string {
	if r, ok := f[code]; ok {
		return r
	}
	return "0"
}

func (fixedRates) Rates(context.Context, string) (map[string]decimal.Decimal, error) {
	return nil, errors.New("offline")
}

func fullFill() spacefill.Fill {
	return spacefill.Fill{
		BaseCurrency: "EUR",
		Currencies:   []string{"USD", "gbp"},
		Accounts: []spacefill.Account{
			{Name: "Card", Type: "bank", Currency: "EUR", InitialBalance: decimal.RequireFromString("120.50")},
			{Name: "Cash", Currency: "USD"},
		},
		Categories: []spacefill.Category{{Name: "Food", Type: "expense", Icon: "🍕", Color: "#fff"}},
		Tags:       []string{"Family", "Trip"},
	}
}

func spaceCount(t *testing.T, a *testApp) int {
	t.Helper()
	all, err := a.svc.Spaces.All(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return len(all)
}

func TestCreateSpaceWithNameOnlyEmpty(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()
	owner := a.createUser("o@test.com", "secret1", auth.RoleUser)

	space, err := a.spaceCreator().Create(ctx, "Plain", owner, spacefill.Fill{})
	if err != nil {
		t.Fatal(err)
	}
	if space.Role != spaces.RoleAdmin {
		t.Fatalf("role %q", space.Role)
	}
	d, _ := a.store.Space(ctx, space.ID)
	if cur, _ := (ledger.Currencies{DB: d}).All(ctx); len(cur) != 0 {
		t.Fatalf("currencies %v", cur)
	}
	if cats, _ := (ledger.Categories{DB: d}).All(ctx, ""); len(cats) != 0 {
		t.Fatalf("categories %v", cats)
	}
}

func TestCreateSpaceNameOnlyOverHTTP(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("o@test.com", "secret1", auth.RoleUser), false)
	res := a.do("POST", "/api/spaces", map[string]any{"name": "Plain"}, sess.Token, sess.CSRF)
	status(t, res, 201, "name only")
}

func TestCreateSpaceFilledOverHTTP(t *testing.T) {
	a := newTestApp(t, WithRates(fixedRates{"USD": "0.9", "EUR": "1", "GBP": "1.2"}))
	sess := a.issue(a.createUser("o@test.com", "secret1", auth.RoleUser), false)
	res := a.do("POST", "/api/spaces", map[string]any{
		"name": "Home", "baseCurrency": "EUR", "currencies": []string{"USD"},
		"accounts": []map[string]any{{"name": "Card", "currency": "EUR", "initialBalance": 10.25}},
		"tags":     []string{"Family"},
	}, sess.Token, sess.CSRF)
	if res.StatusCode != 201 {
		t.Fatalf("filled: %d %v", res.StatusCode, decodeJSON(t, res))
	}
	id := int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	list := dataList(t, decodeJSON(t, a.inSpace(id, "GET", "/api/accounts", nil, sess)))
	if len(list) != 1 {
		t.Fatalf("accounts %v", list)
	}
}

func TestCreateSpaceFilledBuildsEverything(t *testing.T) {
	a := newTestApp(t)
	a.svc.Rates = fixedRates{"USD": "0.9", "EUR": "1", "GBP": "1.2"}
	ctx := context.Background()
	owner := a.createUser("o@test.com", "secret1", auth.RoleUser)

	space, err := a.spaceCreator().Create(ctx, "Home", owner, fullFill())
	if err != nil {
		t.Fatal(err)
	}
	d, _ := a.store.Space(ctx, space.ID)

	curs := ledger.Currencies{DB: d}
	base, _ := curs.Base(ctx)
	if base == nil || base.Code != "EUR" || !base.Rate.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("base %+v", base)
	}
	all, _ := curs.All(ctx)
	if len(all) != 3 || all[0].Code != "EUR" {
		t.Fatalf("base must be created first, got %+v", all)
	}
	usd, _ := curs.ByCode(ctx, "USD")
	gbp, _ := curs.ByCode(ctx, "GBP")
	if usd == nil || !usd.Rate.Equal(decimal.RequireFromString("0.9")) || gbp == nil ||
		!gbp.Rate.Equal(decimal.RequireFromString("1.2")) {
		t.Fatalf("rates from the source: usd=%+v gbp=%+v", usd, gbp)
	}

	accs, _ := (ledger.Accounts{DB: d}).All(ctx, false, false)
	if len(accs) != 2 {
		t.Fatalf("accounts %v", accs)
	}
	byName := map[string]ledger.Account{}
	for _, acc := range accs {
		byName[acc.Name] = acc
	}
	card := byName["Card"]
	if card.Type != "bank" || card.Currency.Code != "EUR" ||
		!card.InitialBalance.Decimal().Equal(decimal.RequireFromString("120.5")) ||
		!card.Balance.Decimal().Equal(decimal.RequireFromString("120.5")) {
		t.Fatalf("card %+v", card)
	}
	if cash := byName["Cash"]; cash.Type != "cash" || cash.Currency.Code != "USD" || !cash.Balance.Decimal().IsZero() {
		t.Fatalf("cash %+v", cash)
	}

	cats, _ := (ledger.Categories{DB: d}).All(ctx, "")
	defaults := map[string]string{}
	names := map[string]bool{}
	for _, c := range cats {
		names[c.Name] = true
		if c.IsDefault {
			defaults[c.Type] = c.Name
		}
	}
	if !names["Food"] || defaults["expense"] != "#OTHER" || defaults["income"] != "#OTHER_INCOME" || len(defaults) != 2 {
		t.Fatalf("categories %v defaults %v", names, defaults)
	}

	tags, _ := (ledger.Tags{DB: d}).All(ctx)
	if len(tags) != 2 {
		t.Fatalf("tags %v", tags)
	}
}

func TestCreateSpaceKeepsDefaultsRequestNames(t *testing.T) {
	a := newTestApp(t)
	a.svc.Rates = fixedRates{"EUR": "1"}
	ctx := context.Background()
	owner := a.createUser("o@test.com", "secret1", auth.RoleUser)

	fill := spacefill.Fill{BaseCurrency: "EUR", Accounts: []spacefill.Account{{Name: "Wallet", Currency: "EUR"}},
		Categories: []spacefill.Category{
			{Name: "Misc", Type: "expense", IsDefault: true},
			{Name: "Pay", Type: "income"},
		}}
	space, err := a.spaceCreator().Create(ctx, "S", owner, fill)
	if err != nil {
		t.Fatal(err)
	}
	d, _ := a.store.Space(ctx, space.ID)
	cats, _ := (ledger.Categories{DB: d}).All(ctx, "")
	defaults := map[string]string{}
	for _, c := range cats {
		if c.IsDefault {
			defaults[c.Type] = c.Name
		}
	}
	if defaults["expense"] != "Misc" || defaults["income"] != "#OTHER_INCOME" {
		t.Fatalf("defaults %v", defaults)
	}
}

func TestCreateSpaceRefusesBadInputEarly(t *testing.T) {
	a := newTestApp(t)
	a.svc.Rates = fixedRates{"USD": "1", "EUR": "1"}
	owner := a.createUser("o@test.com", "secret1", auth.RoleUser)
	before := spaceCount(t, a)

	bad := map[string]spacefill.Fill{
		"currencies without a base": {Currencies: []string{"USD"}},
		"account in a currency the space lacks": {BaseCurrency: "EUR",
			Accounts: []spacefill.Account{{Name: "A", Currency: "USD"}}},
		"account without a name": {BaseCurrency: "EUR", Accounts: []spacefill.Account{{Currency: "EUR"}}},
		"category of a bad type": {BaseCurrency: "EUR", Categories: []spacefill.Category{{Name: "X", Type: "transfer"}}},
		"two expense defaults": {BaseCurrency: "EUR", Categories: []spacefill.Category{
			{Name: "A", Type: "expense", IsDefault: true}, {Name: "B", Type: "expense", IsDefault: true}}},
		"blank tag":      {BaseCurrency: "EUR", Tags: []string{" "}},
		"malformed code": {BaseCurrency: "E/R"},
	}
	for name, fill := range bad {
		var fe *spacefill.Error
		if _, err := a.spaceCreator().Create(context.Background(), "S", owner, fill); !errors.As(err, &fe) {
			t.Errorf("%s: got %v, want a spacefill.Error", name, err)
		}
	}
	if got := spaceCount(t, a); got != before {
		t.Fatalf("spaces %d, was %d", got, before)
	}
}

func TestCreateSpaceEnforcesGuestLimit(t *testing.T) {
	a := newTestApp(t)
	a.svc.Rates = fixedRates{"EUR": "1"}
	ctx := context.Background()
	fill := spacefill.Fill{BaseCurrency: "EUR", Accounts: []spacefill.Account{{Name: "Wallet", Currency: "EUR"}}}

	guest := a.createUser("g@test.com", "secret1", auth.RoleGuest)
	if _, err := a.spaceCreator().Create(ctx, "G", guest, fill); !errors.Is(err, spaces.ErrGuestCannotAdmin) {
		t.Fatalf("guest: %v", err)
	}

	user := a.createUser("u@test.com", "secret1", auth.RoleUser)
	_ = a.svc.Settings.Set(ctx, "max_spaces_per_user", 1)
	if _, err := a.spaceCreator().Create(ctx, "One", user, fill); err != nil {
		t.Fatal(err)
	}
	before := spaceCount(t, a)
	if _, err := a.spaceCreator().Create(ctx, "Two", user, fill); !errors.Is(err, spaces.ErrSpaceLimit) {
		t.Fatalf("limit: %v", err)
	}
	if got := spaceCount(t, a); got != before {
		t.Fatalf("spaces %d, was %d", got, before)
	}
}

func TestCreateSpaceFailureLeavesNothing(t *testing.T) {
	a := newTestApp(t)
	// ZZZ is well formed but not in the catalog: it fails after the base exists.
	a.svc.Rates = fixedRates{"USD": "1", "EUR": "1"}
	ctx := context.Background()
	owner := a.createUser("o@test.com", "secret1", auth.RoleUser)
	before, _ := a.svc.Spaces.All(ctx)

	fill := spacefill.Fill{BaseCurrency: "EUR", Currencies: []string{"USD", "ZZZ"}}
	if _, err := a.spaceCreator().Create(ctx, "Broken", owner, fill); err == nil {
		t.Fatal("want an error")
	}
	after, _ := a.svc.Spaces.All(ctx)
	if len(after) != len(before) {
		t.Fatalf("registry: %d spaces, was %d", len(after), len(before))
	}
	known := map[int64]bool{}
	for _, s := range before {
		known[s.ID] = true
	}
	// The failed space took the next id; its file must be gone too.
	for id := int64(1); id <= int64(len(before))+2; id++ {
		if known[id] {
			continue
		}
		if _, err := a.store.Space(ctx, id); err == nil {
			t.Fatalf("space file %d still exists", id)
		}
	}
	mine, _ := a.svc.Spaces.ForUser(ctx, owner.ID)
	if len(mine) != 0 {
		t.Fatalf("owner still has %v", mine)
	}
}

func TestCreateSpaceRefusesCurrencyWithoutRate(t *testing.T) {
	a := newTestApp(t)
	a.svc.Rates = fixedRates{"EUR": "1", "USD": "1.1", "JPY": "0"}
	ctx := context.Background()
	owner := a.createUser("o@test.com", "secret1", auth.RoleUser)
	before := spaceCount(t, a)

	_, err := a.spaceCreator().Create(ctx, "Yen", owner, spacefill.Fill{BaseCurrency: "EUR", Currencies: []string{"JPY"},
		Accounts: []spacefill.Account{{Name: "Wallet", Currency: "EUR"}}})
	var fe *spacefill.Error
	if !errors.As(err, &fe) || fe.Field != "currencies" {
		t.Fatalf("got %v, want a spacefill.Error on currencies", err)
	}
	if got := spaceCount(t, a); got != before {
		t.Fatalf("spaces %d, was %d", got, before)
	}

	// A base without a known rate is fine: its rate is 1 by definition.
	a.svc.Rates = fixedRates{"JPY": "0", "EUR": "0.006"}
	if _,
		err := a.spaceCreator().Create(ctx, "Yen base", owner,
		spacefill.Fill{BaseCurrency: "JPY", Currencies: []string{"EUR"},
			Accounts: []spacefill.Account{{Name: "Wallet", Currency: "JPY"}}}); err != nil {
		t.Fatal(err)
	}
}

func TestCreateSpaceNeedsAnAccount(t *testing.T) {
	a := newTestApp(t)
	a.svc.Rates = fixedRates{"EUR": "1"}
	owner := a.createUser("o@test.com", "secret1", auth.RoleUser)
	before := spaceCount(t, a)

	var fe *spacefill.Error
	_, err := a.spaceCreator().Create(context.Background(), "S", owner, spacefill.Fill{BaseCurrency: "EUR"})
	if !errors.As(err, &fe) || fe.Field != "accounts" {
		t.Fatalf("got %v, want a spacefill.Error on accounts", err)
	}
	if got := spaceCount(t, a); got != before {
		t.Fatalf("spaces %d, was %d", got, before)
	}

	one := spacefill.Fill{BaseCurrency: "EUR", Accounts: []spacefill.Account{{Name: "Wallet", Currency: "EUR"}}}
	if _, err := a.spaceCreator().Create(context.Background(), "S", owner, one); err != nil {
		t.Fatalf("one account: %v", err)
	}
}
