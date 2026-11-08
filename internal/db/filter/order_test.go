package filter

import (
	"context"
	"slices"
	"testing"

	"savvy-go/internal/db/dbtest"
)

// Ids 1..6: 1 dated 06-01, 2 dated 06-01 with day_order 0, 3 undated,
// 4 dated 05-01, 5 undated, 6 dated 06-01.
func TestListTransactionsDateOrder(t *testing.T) {
	d := dbtest.Space(t)
	ctx := context.Background()
	for _, q := range []string{
		`INSERT INTO transactions(type, date) VALUES ('expense','2025-06-01')`,
		`INSERT INTO transactions(type, date, day_order) VALUES ('expense','2025-06-01', 0)`,
		`INSERT INTO transactions(type, date) VALUES ('expense', NULL)`,
		`INSERT INTO transactions(type, date) VALUES ('expense','2025-05-01')`,
		`INSERT INTO transactions(type, date) VALUES ('expense', NULL)`,
		`INSERT INTO transactions(type, date) VALUES ('expense','2025-06-01')`,
	} {
		if _, err := d.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}

	cases := map[string]struct {
		dir  string
		want []int64
	}{
		// Within a day: COALESCE(day_order, id), so id 2 (order 0) comes first ascending.
		"asc":  {"asc", []int64{4, 2, 1, 6, 3, 5}},
		"desc": {"desc", []int64{6, 1, 2, 4, 5, 3}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			rows, err := ListTransactions(ctx, d, TxFilter{SortBy: "date", SortDir: c.dir}, 10, 0)
			if err != nil {
				t.Fatal(err)
			}
			var got []int64
			for _, r := range rows {
				got = append(got, r.ID)
			}
			if !slices.Equal(got, c.want) {
				t.Fatalf("order = %v, want %v", got, c.want)
			}
			// A later page must continue the same order.
			rest, err := ListTransactions(ctx, d, TxFilter{SortBy: "date", SortDir: c.dir}, 10, 4)
			if err != nil || len(rest) != 2 || rest[0].ID != c.want[4] {
				t.Fatalf("offset page = %v, %v", rest, err)
			}
		})
	}
}
