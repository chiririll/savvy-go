package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/money"
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
			"code": c.Code, "name": c.Code, "symbol": c.Code, "decimals": c.Decimals, "rate": num(c.RateIn(moneytest.Currencies[0])),
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

func TestCurrencyAPIAcceptsEveryScaleAndRejectsOutOfRange(t *testing.T) {
	e := newWireEnv(t) // creating every currency, zero and twelve decimals included, is the first check
	post := func(body map[string]any) (int, map[string]any) {
		res := e.a.do("POST", "/api/currencies", body, e.sess.Token, e.sess.CSRF)
		out := decodeJSON(t, res)
		return res.StatusCode, out
	}
	for _, c := range []struct {
		name  string
		body  map[string]any
		field string // the validation error expected, "" for success
	}{
		{"decimals given as zero", map[string]any{"code": "KRW", "name": "Won", "symbol": "W", "decimals": 0, "rate": 0.00073}, ""},
		{"decimals thirteen", map[string]any{"code": "TOO", "name": "x", "symbol": "x", "decimals": 13, "rate": 1}, "decimals"},
		{"rate below the minimum", map[string]any{"code": "LOW", "name": "x", "symbol": "x", "decimals": 2, "rate": json.Number("0.0000000000000009")}, "rate"},
		{"rate above the maximum", map[string]any{"code": "HIG", "name": "x", "symbol": "x", "decimals": 2, "rate": json.Number("1000000000000001")}, "rate"},
		{"negative rate", map[string]any{"code": "NEG", "name": "x", "symbol": "x", "decimals": 2, "rate": -2}, "rate"},
	} {
		status, body := post(c.body)
		if c.field == "" {
			if status != http.StatusCreated {
				t.Errorf("%s: %d %v", c.name, status, body)
			}
			continue
		}
		errs, _ := body["errors"].(map[string]any)
		if status != http.StatusUnprocessableEntity || errs[c.field] == nil {
			t.Errorf("%s: %d %v, want a 422 on %q", c.name, status, body, c.field)
		}
	}
	// A currency given without decimals keeps the old default.
	status, body := post(map[string]any{"code": "DEF", "name": "x", "symbol": "x"})
	if status != http.StatusCreated || body["data"].(map[string]any)["decimals"].(float64) != 2 {
		t.Errorf("absent decimals: %d %v", status, body)
	}
}

func TestAmountsOnTheWireInEveryCurrency(t *testing.T) {
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
			if body, want := rawBody(t, res), `"currentBalance":`+canonical(balance, c.Decimals)+`,`; !strings.Contains(body, want) {
				t.Fatalf("account %s does not contain %s", body, want)
			}
		})
	})
}

func TestOutOfRangeAmountsAreRefusedOnTheWire(t *testing.T) {
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

func TestConvertEndpointForEveryPair(t *testing.T) {
	e := newWireEnv(t)
	moneytest.ForEachPair(t, func(t *testing.T, from, to moneytest.Currency) {
		moneytest.ForEachAmount(t, moneytest.Valid(from.Decimals), func(t *testing.T, a moneytest.Amount) {
			res := e.a.do("POST", "/api/currencies/convert", map[string]any{
				"amount": num(a.Input), "from_currency_id": e.cur[from.Code], "to_currency_id": e.cur[to.Code],
			}, e.sess.Token, e.sess.CSRF)
			exact := moneytest.Converted(decimal.New(a.Minor, -int32(from.Decimals)), from, to)
			want, err := money.FromInput(exact, money.Unit{Decimals: to.Decimals})
			if err != nil { // the result does not fit
				if res.StatusCode != http.StatusUnprocessableEntity {
					t.Fatalf("status %d, want 422 for a result of %s", res.StatusCode, exact)
				}
				res.Body.Close()
				return
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("status %d: %s", res.StatusCode, rawBody(t, res))
			}
			if body, wantJSON := rawBody(t, res), `"result":`+canonical(want.Minor(), to.Decimals); !strings.Contains(body, wantJSON) {
				t.Fatalf("%s does not contain %s", body, wantJSON)
			}
		})
	})
}
