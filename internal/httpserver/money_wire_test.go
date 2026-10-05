package httpserver

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func rawBody(t *testing.T, res *http.Response) string {
	t.Helper()
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMoneyIsSerializedAsNumbersAtCurrencyScale(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("wire@test.com", "secret1", roleEditor)
	sess := a.issue(u, false)

	mkCurrency := func(code string, decimals int, rate any, base bool) int64 {
		res := a.do("POST", "/api/currencies", map[string]any{
			"code": code, "name": code, "symbol": code, "decimals": decimals, "is_base": base, "rate": rate,
		}, sess.Token, sess.CSRF)
		return int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	}
	usd := mkCurrency("USD", 2, 1, true)
	jpy := mkCurrency("JPY", 0, 0.01, false)
	btc := mkCurrency("BTC", 8, 50000, false)

	mkAccount := func(name string, cur int64, initial any) int64 {
		res := a.do("POST", "/api/accounts", map[string]any{
			"name": name, "type": "cash", "currency_id": cur, "initial_balance": initial,
		}, sess.Token, sess.CSRF)
		return int64(decodeJSON(t, res)["data"].(map[string]any)["id"].(float64))
	}
	usdAcc := mkAccount("USD cash", usd, 100)
	jpyAcc := mkAccount("JPY cash", jpy, 3000)
	btcAcc := mkAccount("BTC wallet", btc, 0)

	postTx := func(acc int64, amount any) string {
		res := a.do("POST", "/api/transactions", map[string]any{
			"type": "income", "account_id": acc, "amount": amount, "date": "2024-01-15",
		}, sess.Token, sess.CSRF)
		if res.StatusCode != 201 {
			t.Fatalf("create tx status %d", res.StatusCode)
		}
		return rawBody(t, res)
	}

	cases := []struct {
		name   string
		acc    int64
		amount any
		want   string
	}{
		{"usd is a plain number", usdAcc, 12.5, `"amount":12.5,`},
		{"usd rounds half away from zero", usdAcc, 10.005, `"amount":10.01`},
		{"jpy has no decimals", jpyAcc, 1500, `"amount":1500,`},
		{"btc keeps eight decimals", btcAcc, 0.00012345, `"amount":0.00012345,`},
		{"btc rounds to eight decimals", btcAcc, 0.000000015, `"amount":0.00000002,`},
	}
	for _, c := range cases {
		if body := postTx(c.acc, c.amount); !strings.Contains(body, c.want) {
			t.Errorf("%s: response %s does not contain %s", c.name, body, c.want)
		}
	}

	res := a.do("GET", "/api/accounts/"+itoa(usdAcc), nil, sess.Token, "")
	body := rawBody(t, res)
	for _, want := range []string{`"initialBalance":100,`, `"currentBalance":122.51,`} {
		if !strings.Contains(body, want) {
			t.Errorf("account response %s does not contain %s", body, want)
		}
	}

	res = a.do("GET", "/api/currencies", nil, sess.Token, "")
	body = rawBody(t, res)
	for _, want := range []string{`"rate":0.01`, `"rate":50000`} {
		if !strings.Contains(body, want) {
			t.Errorf("currencies response %s does not contain %s", body, want)
		}
	}
	if strings.Contains(body, `"rate":"`) {
		t.Errorf("rate must be a JSON number, got %s", body)
	}
}
