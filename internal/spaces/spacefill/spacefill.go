// Package spacefill creates a space together with its first content: currencies,
// accounts, categories and tags, all or nothing.
package spacefill

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/ledger"
	"savvy-go/internal/seed"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// maxFillItems bounds each list of a create-space request.
const maxFillItems = 200

var currencyCodePattern = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)

// Fill is the optional content of POST /api/spaces beside the name.
type Fill struct {
	Emoji        *string    `json:"emoji"`
	Color        *string    `json:"color"`
	BaseCurrency string     `json:"baseCurrency"`
	Currencies   []string   `json:"currencies"`
	Accounts     []Account  `json:"accounts"`
	Categories   []Category `json:"categories"`
	Tags         []string   `json:"tags"`
}

// Account is an account to open in the new space.
type Account struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Currency       string          `json:"currency"`
	InitialBalance decimal.Decimal `json:"initialBalance"`
}

// Category is a category to create in the new space.
type Category struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	IsDefault bool   `json:"isDefault"`
}

// Error is a problem with the request body, reported as a 422 on field.
type Error struct{ Field, Msg string }

func (e *Error) Error() string { return e.Field + ": " + e.Msg }

func invalid(field, format string, args ...any) error {
	return &Error{Field: field, Msg: fmt.Sprintf(format, args...)}
}

func (f Fill) empty() bool {
	return f.BaseCurrency == "" && len(f.Currencies) == 0 && len(f.Accounts) == 0 && len(f.Categories) == 0 &&
		len(f.Tags) == 0
}

func normalizeCode(raw string) string { return strings.ToUpper(strings.TrimSpace(raw)) }

// normalize checks the request without touching any storage and returns it
// with codes upper-cased and trimmed, and the two default categories added when
// it names none. Currencies lists the base first, once each.
func (f Fill) normalize() (Fill, error) {
	if len(f.Currencies) > maxFillItems || len(f.Accounts) > maxFillItems || len(f.Categories) > maxFillItems ||
		len(f.Tags) > maxFillItems {
		return f, invalid("body", "Too many items.")
	}
	f.BaseCurrency = normalizeCode(f.BaseCurrency)
	if f.BaseCurrency == "" {
		return f, invalid("baseCurrency", "The base currency is required.")
	}
	owned := map[string]bool{}
	var codes []string
	for _, raw := range append([]string{f.BaseCurrency}, f.Currencies...) {
		code := normalizeCode(raw)
		if !currencyCodePattern.MatchString(code) {
			return f, invalid("currencies", "%q is not a currency code.", raw)
		}
		if !owned[code] {
			owned[code] = true
			codes = append(codes, code)
		}
	}
	f.Currencies = codes

	accounts := make([]Account, len(f.Accounts))
	for i, a := range f.Accounts {
		a.Name, a.Currency = strings.TrimSpace(a.Name), normalizeCode(a.Currency)
		if a.Name == "" {
			return f, invalid("accounts", "Every account needs a name.")
		}
		if !owned[a.Currency] {
			return f, invalid("accounts", "Account %q uses %q, which is not among the space's currencies.", a.Name, a.Currency)
		}
		accounts[i] = a
	}
	f.Accounts = accounts

	cats, err := withDefaultCategories(f.Categories)
	if err != nil {
		return f, err
	}
	f.Categories = cats

	tags := make([]string, len(f.Tags))
	for i, t := range f.Tags {
		if tags[i] = strings.TrimSpace(t); tags[i] == "" {
			return f, invalid("tags", "A tag needs a name.")
		}
	}
	f.Tags = tags
	return f, nil
}

// withDefaultCategories validates the categories and adds the preset default of
// each type the request gives none (a type needs exactly one).
func withDefaultCategories(in []Category) ([]Category, error) {
	out := make([]Category, 0, len(in)+2)
	hasDefault := map[string]bool{}
	for _, c := range in {
		c.Name = strings.TrimSpace(c.Name)
		if c.Name == "" {
			return nil, invalid("categories", "Every category needs a name.")
		}
		if c.Type != "expense" && c.Type != "income" {
			return nil, invalid("categories", "Category %q must be of type expense or income.", c.Name)
		}
		if c.IsDefault {
			if hasDefault[c.Type] {
				return nil, invalid("categories", "Only one %s category can be the default.", c.Type)
			}
			hasDefault[c.Type] = true
		}
		out = append(out, c)
	}
	for _, p := range seed.DefaultCategories() {
		if p.IsDefault && !hasDefault[p.Type] {
			out = append(out, Category{Name: p.Name, Type: p.Type, Icon: p.Icon, Color: p.Color, IsDefault: true})
		}
	}
	return out, nil
}

// Creator creates spaces. Rates is the currency data of the server.
type Creator struct {
	Spaces spaces.Service
	Store  store.Store
	Rates  ledger.RateSource
}

// priceCurrencies looks every code up in the server's currency data, once, and
// returns them in the order given with their rates in the base. A foreign
// currency without a known rate is refused: storing the rate 1 would convert
// every amount wrongly until the next update, and there is no way to tell it
// apart from a real 1. The base needs no rate.
func (cr Creator) priceCurrencies(ctx context.Context, base string, codes []string) ([]ledger.Currency, error) {
	if cr.Rates == nil {
		return nil, invalid("currencies", "Currency data is unavailable.")
	}
	catalog := map[string]ledger.Currency{}
	for _, c := range cr.Rates.Catalog(ctx, base) {
		catalog[strings.ToUpper(c.Code)] = c
	}
	out := make([]ledger.Currency, len(codes))
	for i, code := range codes {
		c, ok := catalog[code]
		if !ok {
			return nil, invalid("currencies", "Unknown currency %q.", code)
		}
		c.IsBase = code == base
		switch {
		case c.IsBase:
			c.Rate = decimal.NewFromInt(1)
		case !c.Rate.IsPositive():
			return nil, invalid(
				"currencies",
				"No exchange rate for %s is available right now. Try again later or leave it out.",
				code,
			)
		}
		out[i] = c
	}
	return out, nil
}

// Create creates a space for owner and, when fill names anything, fills it
// in the same call: currencies (base first), accounts, categories, tags. The
// space limit and the guest check come from spaces.CreateOwned. If filling
// fails the space is discarded, so nothing of it remains. A fill problem is a
// *Error.
func (cr Creator) Create(ctx context.Context, name string, owner *auth.User, fill Fill) (*spaces.Space, error) {
	look := spaces.Look{Emoji: fill.Emoji, Color: fill.Color}
	if fill.empty() {
		return cr.Spaces.CreateOwned(ctx, name, owner, look)
	}
	fill, err := fill.normalize()
	if err != nil {
		return nil, err
	}
	space, err := cr.Spaces.CreateOwned(ctx, name, owner, look)
	if err != nil {
		return nil, err
	}
	if err := cr.fillSpace(ctx, space.ID, fill); err != nil {
		cr.Spaces.Discard(ctx, space.ID)
		return nil, err
	}
	return space, nil
}

// fillSpace writes a normalized fill into space id in one transaction.
func (cr Creator) fillSpace(ctx context.Context, id int64, fill Fill) error {
	currencies, err := cr.priceCurrencies(ctx, fill.BaseCurrency, fill.Currencies)
	if err != nil {
		return err
	}
	db, err := cr.Store.Space(ctx, id)
	if err != nil {
		return err
	}
	return store.Tx(ctx, db, func(tx store.DB) error {
		byCode := map[string]*ledger.Currency{}
		for _, c := range currencies { // the base is first, so it is the first row and the base
			created, err := (ledger.Currencies{DB: tx, Source: cr.Rates}).Create(ctx, c)
			if err != nil {
				return invalid("currencies", "Currency %s: %v.", c.Code, err)
			}
			byCode[created.Code] = created
		}
		accounts := ledger.Accounts{DB: tx}
		for _, a := range fill.Accounts {
			if _, err := accounts.Create(ctx, ledger.AccountInput{
				Name: a.Name, Type: a.Type, CurrencyID: byCode[a.Currency].ID,
				InitialBalance: a.InitialBalance, IsActive: true,
			}); err != nil {
				return invalid("accounts", "Account %q: %v.", a.Name, err)
			}
		}
		cats := ledger.Categories{DB: tx}
		for _, c := range fill.Categories {
			cat := ledger.Category{Name: c.Name, Type: c.Type, IsDefault: c.IsDefault}
			if icon := c.Icon; icon != "" {
				cat.Icon = &icon
			}
			if color := c.Color; color != "" {
				cat.Color = &color
			}
			if _, err := cats.Create(ctx, cat); err != nil {
				return invalid("categories", "Category %q: %v.", c.Name, err)
			}
		}
		tags := ledger.Tags{DB: tx}
		for _, t := range fill.Tags {
			if _, err := tags.Create(ctx, t); err != nil {
				return invalid("tags", "Tag %q: %v.", t, err)
			}
		}
		return nil
	})
}
