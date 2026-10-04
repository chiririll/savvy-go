package money

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
)

var (
	usd = Unit{ID: 1, Decimals: 2}
	jpy = Unit{ID: 2, Decimals: 0}
	btc = Unit{ID: 3, Decimals: 8}
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestRoundTrip(t *testing.T) {
	cases := []struct {
		in    string
		unit  Unit
		minor int64
	}{
		{"123.45", usd, 12345},
		{"-0.10", usd, -10},
		{"1500", jpy, 1500},
		{"0.00012345", btc, 12345},
		{"0.1", usd, 10},
	}
	for _, c := range cases {
		m := FromDecimal(dec(c.in), c.unit)
		if m.Minor() != c.minor {
			t.Errorf("FromDecimal(%s,%+v)=%d want %d", c.in, c.unit, m.Minor(), c.minor)
		}
		if !m.Decimal().Equal(dec(c.in)) {
			t.Errorf("Decimal of %s = %s", c.in, m.Decimal())
		}
	}
}

func TestFromDecimalRoundsHalfAwayFromZero(t *testing.T) {
	for _, c := range []struct {
		in    string
		unit  Unit
		minor int64
	}{{"0.005", usd, 1}, {"-0.005", usd, -1}, {"2.5", jpy, 3}, {"0.004", usd, 0}} {
		if got := FromDecimal(dec(c.in), c.unit).Minor(); got != c.minor {
			t.Errorf("%s -> %d want %d", c.in, got, c.minor)
		}
	}
}

func TestSumIsExact(t *testing.T) {
	sum := New(10, usd).Add(New(20, usd))
	if !sum.Decimal().Equal(dec("0.3")) {
		t.Fatalf("0.10+0.20=%s", sum)
	}
}

func TestMixedCurrenciesPanic(t *testing.T) {
	for name, op := range map[string]func(){
		"add": func() { New(1, usd).Add(New(1, jpy)) },
		"sub": func() { New(1, usd).Sub(New(1, btc)) },
		"cmp": func() { New(1, usd).Cmp(New(1, jpy)) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected a panic for mixed currencies", name)
				}
			}()
			op()
		}()
	}
}

func TestEqualRequiresSameCurrency(t *testing.T) {
	if New(100, usd).Equal(New(100, jpy)) {
		t.Error("same minor units in different currencies are not equal")
	}
	if !New(100, usd).Equal(New(100, usd)) {
		t.Error("equal amounts reported unequal")
	}
}

func TestNullVariants(t *testing.T) {
	if FromNullMinor(sql.NullInt64{}, usd) != nil {
		t.Fatal("null should map to nil")
	}
	d := dec("1.25")
	m := FromNullDecimal(&d, usd)
	if n := ToNullMinor(m); !n.Valid || n.Int64 != 125 {
		t.Fatalf("got %+v", n)
	}
	if ToNullMinor(nil).Valid {
		t.Fatal("nil should map to invalid")
	}
	if FromNullDecimal(nil, usd) != nil {
		t.Fatal("nil should stay nil")
	}
}

func TestMarshalJSONIsANumber(t *testing.T) {
	out, err := json.Marshal(map[string]Money{"a": New(1250, usd), "b": New(1500, jpy), "c": New(12345, btc)})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"a":12.5,"b":1500,"c":0.00012345}` {
		t.Fatalf("got %s", out)
	}
}
