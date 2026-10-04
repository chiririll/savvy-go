package money

import (
	"database/sql"
	"testing"

	"github.com/shopspring/decimal"
)

func TestRoundTrip(t *testing.T) {
	cases := []struct {
		in       string
		decimals int
		minor    int64
	}{
		{"123.45", 2, 12345},
		{"-0.10", 2, -10},
		{"1500", 0, 1500},
		{"0.00012345", 8, 12345},
		{"0.1", 2, 10},
	}
	for _, c := range cases {
		d := decimal.RequireFromString(c.in)
		if got := ToMinor(d, c.decimals); got != c.minor {
			t.Errorf("ToMinor(%s,%d)=%d want %d", c.in, c.decimals, got, c.minor)
		}
		if back := FromMinor(c.minor, c.decimals); !back.Equal(d) {
			t.Errorf("FromMinor(%d,%d)=%s want %s", c.minor, c.decimals, back, d)
		}
	}
}

func TestToMinorRoundsHalfAwayFromZero(t *testing.T) {
	if got := ToMinor(decimal.RequireFromString("0.005"), 2); got != 1 {
		t.Errorf("0.005 -> %d want 1", got)
	}
	if got := ToMinor(decimal.RequireFromString("-0.005"), 2); got != -1 {
		t.Errorf("-0.005 -> %d want -1", got)
	}
	if got := ToMinor(decimal.RequireFromString("2.5"), 0); got != 3 {
		t.Errorf("2.5 -> %d want 3", got)
	}
}

func TestFloatSumIsExact(t *testing.T) {
	sum := FromMinor(10, 2).Add(FromMinor(20, 2))
	if !sum.Equal(decimal.RequireFromString("0.3")) {
		t.Fatalf("0.10+0.20=%s", sum)
	}
}

func TestNullVariants(t *testing.T) {
	if FromNullMinor(sql.NullInt64{}, 2) != nil {
		t.Fatal("null should map to nil")
	}
	d := decimal.RequireFromString("1.25")
	n := ToNullMinor(&d, 2)
	if !n.Valid || n.Int64 != 125 {
		t.Fatalf("got %+v", n)
	}
	if ToNullMinor(nil, 2).Valid {
		t.Fatal("nil should map to invalid")
	}
}
