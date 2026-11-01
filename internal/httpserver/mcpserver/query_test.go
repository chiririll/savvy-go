package mcpserver

import (
	"net/url"
	"testing"
)

// These helpers build the query of the REST call a tool makes, so what they let
// through is what reaches the route.
func TestQueryHelpersRefuseBadDatesValues(t *testing.T) {
	q := url.Values{}
	if err := setDate(q, "start_date", "2026-02-30x"); err == nil {
		t.Error("a malformed date was accepted")
	}
	if err := setDate(q, "start_date", "2026-02-01&x=1"); err == nil {
		t.Error("a date carrying another parameter was accepted")
	}
	if err := setEnum(q, "type", "expense&x=1", "expense", "income"); err == nil {
		t.Error("a value outside the allowed set was accepted")
	}
	if len(q) != 0 {
		t.Errorf("refused values reached the query: %v", q)
	}

	if err := firstErr(nil, setDate(q, "end_date", "2026-02-01"),
		setEnum(q, "type", "income", "expense", "income")); err != nil {
		t.Fatal(err)
	}
	if q.Get("end_date") != "2026-02-01" || q.Get("type") != "income" {
		t.Errorf("query = %v", q)
	}

	// Empty means not given.
	empty := url.Values{}
	if err := firstErr(setDate(empty, "d", ""), setEnum(empty, "e", "", "a")); err != nil || len(empty) != 0 {
		t.Errorf("empty values: %v %v", err, empty)
	}
}

func TestTransactionsQueryPassesTheSearchText(t *testing.T) {
	q, err := transactionsQuery(transactionsArgs{Query: "armchair & 1 879"})
	if err != nil {
		t.Fatal(err)
	}
	if q.Get("q") != "armchair & 1 879" || len(q["q"]) != 1 {
		t.Errorf("q = %q", q["q"])
	}
	q, _ = transactionsQuery(transactionsArgs{})
	if _, ok := q["q"]; ok {
		t.Errorf("an empty search reached the query: %v", q)
	}
}

func TestQueryIDsSkipUnsetOnes(t *testing.T) {
	q := url.Values{}
	setID(q, "account_id", 0)
	setID(q, "category_id", -3)
	setIDs(q, "tag_ids", []int64{0, -1})
	if len(q) != 0 {
		t.Fatalf("unset ids reached the query: %v", q)
	}
	setID(q, "account_id", 7)
	setIDs(q, "tag_ids", []int64{3, 0, 5})
	if q.Get("account_id") != "7" || q.Get("tag_ids") != "3,5" {
		t.Errorf("query = %v", q)
	}
}

func TestTransactionsQueryPassesAccountsAndCurrencies(t *testing.T) {
	q, err := transactionsQuery(transactionsArgs{AccountIDs: []int64{1, 0, 2}, Currency: []string{"usd", " ", "JPY"}})
	if err != nil {
		t.Fatal(err)
	}
	if q.Get("account_ids") != "1,2" || q.Get("currency") != "usd,JPY" {
		t.Errorf("account_ids = %q, currency = %q", q.Get("account_ids"), q.Get("currency"))
	}
	q, _ = transactionsQuery(transactionsArgs{})
	if _, ok := q["currency"]; ok {
		t.Errorf("an empty currency list reached the query: %v", q)
	}
}
