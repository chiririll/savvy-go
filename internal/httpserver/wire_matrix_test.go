//go:build integration

package httpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/money/moneytest"
)

// wireEnv is a running app with every moneytest currency created through the
// API, USD (the first) being the base.
type wireEnv struct {
	a    *testApp
	sess *auth.Issued
	cur  map[string]int64 // currency id by code
	acct map[string]int64 // account id by currency code
}

// num is a decimal as a JSON number literal, exactly as written.
func num(d decimal.Decimal) json.Number { return json.Number(d.String()) }

func newWireEnv(t *testing.T) *wireEnv {
	t.Helper()
	a := newTestApp(t)
	u := a.createUser("wire@test.com", "secret1", roleEditor)
	e := &wireEnv{a: a, sess: a.issue(u, false), cur: map[string]int64{}, acct: map[string]int64{}}
	for _, c := range moneytest.Currencies {
		res := a.do("POST", "/api/currencies", map[string]any{
			"code": c.Code, "name": c.Code, "symbol": c.Code, "decimals": c.Decimals,
			"rate": num(c.RateIn(moneytest.Currencies[0])),
		}, e.sess.Token, e.sess.CSRF)
		body := decodeJSON(t, res)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("create %v: %d %v", c, res.StatusCode, body)
		}
		data := body["data"].(map[string]any)
		if int(data["decimals"].(float64)) != c.Decimals {
			t.Fatalf("%v was stored with %v decimals", c, data["decimals"])
		}
		e.cur[c.Code] = int64(data["id"].(float64))
		res = a.do("POST", "/api/accounts", map[string]any{
			"name": c.Code, "type": "cash", "currency_id": e.cur[c.Code], "initial_balance": 0,
		}, e.sess.Token, e.sess.CSRF)
		e.acct[c.Code] = int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	}
	return e
}

func canonical(minor int64, decimals int) string {
	return decimal.New(minor, -int32(decimals)).String()
}

func TestAmountsOnWireInEveryCurrency(t *testing.T) {
	e := newWireEnv(t)
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		balance := int64(0)
		moneytest.ForEachAmount(t, moneytest.Valid(c.Decimals), func(t *testing.T, a moneytest.Amount) {
			res := e.a.do("POST", "/api/transactions", map[string]any{
				"type": "income", "account_id": e.acct[c.Code], "amount": num(a.Input), "date": "2024-01-15",
			}, e.sess.Token, e.sess.CSRF)
			if res.StatusCode != http.StatusCreated {
				t.Fatalf("status %d: %s", res.StatusCode, rawBody(t, res))
			}
			// A plain JSON number at the currency's scale, trailing zeros trimmed.
			if body, want := rawBody(t, res), `"amount":`+canonical(a.Minor, c.Decimals)+`,`; !strings.Contains(body, want) {
				t.Fatalf("response %s does not contain %s", body, want)
			}
			balance += a.Minor
			res = e.a.do("GET", "/api/accounts/"+itoa(e.acct[c.Code]), nil, e.sess.Token, "")
			if body, want := rawBody(t, res), `"currentBalance":`+canonical(balance, c.Decimals)+`,`; !strings.Contains(
				body,
				want,
			) {
				t.Fatalf("account %s does not contain %s", body, want)
			}
		})
	})
}

func TestOutRangeAmountsRefusedOnWire(t *testing.T) {
	e := newWireEnv(t)
	moneytest.ForEach(t, func(t *testing.T, c moneytest.Currency) {
		moneytest.ForEachAmount(t, moneytest.Invalid(c.Decimals), func(t *testing.T, a moneytest.Amount) {
			for name, req := range map[string]func() *http.Response{
				"transaction": func() *http.Response {
					return e.a.do("POST", "/api/transactions", map[string]any{
						"type": "income", "account_id": e.acct[c.Code], "amount": num(a.Input), "date": "2024-01-15",
					}, e.sess.Token, e.sess.CSRF)
				},
				"account": func() *http.Response {
					return e.a.do("POST", "/api/accounts", map[string]any{
						"name": "big " + name(c, a), "type": "cash", "currency_id": e.cur[c.Code], "initial_balance": num(a.Input),
					}, e.sess.Token, e.sess.CSRF)
				},
				"budget": func() *http.Response {
					return e.a.do("POST", "/api/budgets", map[string]any{
						"name": "big", "amount": num(a.Input), "currency_id": e.cur[c.Code], "period": "monthly",
					}, e.sess.Token, e.sess.CSRF)
				},
			} {
				res := req()
				if res.StatusCode != http.StatusUnprocessableEntity {
					t.Errorf("%s: status %d: %s", name, res.StatusCode, rawBody(t, res))
					continue
				}
				res.Body.Close()
			}
			res := e.a.do("GET", "/api/accounts/"+itoa(e.acct[c.Code]), nil, e.sess.Token, "")
			if body, want := rawBody(t, res), `"currentBalance":0,`; !strings.Contains(body, want) {
				t.Errorf("a refused transaction changed the balance: %s", body)
			}
		})
	})
}

func name(c moneytest.Currency, a moneytest.Amount) string { return c.Code + " " + a.Name }

func rawBody(t *testing.T, res *http.Response) string {
	t.Helper()
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// An exchange rate is a JSON number, not a string, and keeps the digits it was given.
func TestCurrencyRateIsAJSONNumber(t *testing.T) {
	a := newTestApp(t)
	sess := a.issue(a.createUser("rate@test.com", "secret1", roleEditor), false)
	for _, c := range []struct {
		code     string
		decimals int
		rate     json.Number
		base     bool
	}{{"USD", 2, "1", true}, {"JPY", 0, "0.01", false}, {"BTC", 8, "50000", false}} {
		res := a.do("POST", "/api/currencies", map[string]any{
			"code": c.code, "name": c.code, "symbol": c.code, "decimals": c.decimals, "is_base": c.base, "rate": c.rate,
		}, sess.Token, sess.CSRF)
		status(t, res, http.StatusCreated, c.code)
	}
	body := rawBody(t, a.do("GET", "/api/currencies", nil, sess.Token, ""))
	for _, want := range []string{`"rate":0.01`, `"rate":50000`} {
		if !strings.Contains(body, want) {
			t.Errorf("currencies response %s does not contain %s", body, want)
		}
	}
	if strings.Contains(body, `"rate":"`) {
		t.Errorf("rate must be a JSON number, got %s", body)
	}
}
