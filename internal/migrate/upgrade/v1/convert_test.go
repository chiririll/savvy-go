package v1_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"savvy-go/internal/db"
	"savvy-go/internal/ledger"
	v1 "savvy-go/internal/migrate/upgrade/v1"
	"savvy-go/internal/recurring"
)

// converted runs the conversion of the fixture and opens the new file.
func converted(t *testing.T) (*sql.DB, string, string) {
	t.Helper()
	src := openV1(t)
	dst := filepath.Join(t.TempDir(), "new.sqlite")
	if err := v1.Convert(context.Background(), src, dst); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	d, err := db.Open(dst)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d, src, dst
}

// rowsOf runs a query and writes each row as its columns joined by "|".
func rowsOf(t *testing.T, d *sql.DB, query string) []string {
	t.Helper()
	rows, err := d.Query(query)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var out []string
	for rows.Next() {
		cells := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range cells {
			ptrs[i] = &cells[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		parts := make([]string, len(cols))
		for i, c := range cells {
			parts[i] = "NULL"
			if c.Valid {
				parts[i] = c.String
			}
		}
		out = append(out, strings.Join(parts, "|"))
	}
	return out
}

func expect(t *testing.T, d *sql.DB, query string, want ...string) {
	t.Helper()
	if got := rowsOf(t, d, query); !reflect.DeepEqual(got, want) {
		t.Errorf("%s\n got: %q\nwant: %q", query, got, want)
	}
}

func TestIsV1TellsTheFilesApart(t *testing.T) {
	d, src, dst := converted(t)
	for path, want := range map[string]bool{src: true, dst: false} {
		got, err := v1.IsV1(context.Background(), path)
		if err != nil || got != want {
			t.Errorf("IsV1(%s) = %v, %v; want %v", filepath.Base(path), got, err, want)
		}
	}
	expect(t, d, `SELECT version FROM schema_migrations`, "space/0001_base")
	expect(t, src2(t, src), `SELECT version FROM schema_migrations`, "space/0001_initial")
}

// src2 opens the v1.0 file again: the conversion must not have changed it.
func src2(t *testing.T, path string) *sql.DB {
	t.Helper()
	d, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestConvertKeepsRowsAndFindsNoOrphans(t *testing.T) {
	d, _, _ := converted(t)
	expect(t, d, `SELECT * FROM pragma_foreign_key_check`)
	expect(t, d, `SELECT id, type, status, external_id FROM transactions ORDER BY id`,
		"1|expense|confirmed|csv:1:abc", "2|income|confirmed|NULL", "3|transfer|confirmed|NULL",
		"4|transfer|confirmed|NULL", "5|expense|confirmed|NULL", "6|debt_lend|confirmed|NULL",
		"7|debt_collection|confirmed|NULL", "8|debt_borrow|confirmed|NULL", "9|debt_payment|confirmed|NULL",
		"10|transfer_out|confirmed|NULL", "11|transfer_in|confirmed|NULL", "12|expense|pending|NULL",
		"13|expense|confirmed|NULL", "14|expense|skipped|NULL")
	expect(t, d, `SELECT transaction_id, tag_id FROM transaction_tag ORDER BY 1, 2`, "1|1", "12|2")
	// Ids are not reused: the sequence of the v1.0 file was 20.
	expect(t, d, `SELECT name, seq FROM sqlite_sequence WHERE name = 'transactions'`, "transactions|20")
}

func TestPaymentsFollowTheMovedMoney(t *testing.T) {
	d, _, _ := converted(t)
	expect(t, d, `SELECT transaction_id, account_id, amount, space_transfer_uuid FROM payments ORDER BY 1, 2`,
		"1|1|-1250|NULL", "2|1|300000|NULL",
		"3|1|-1000|NULL", "3|2|117000|NULL", // a transfer between two accounts
		"4|1|-100|NULL", // from an account to itself: one payment of the net
		"5|1|0|NULL",    // an amount of 0 keeps its payment
		"6|1|-50000|NULL", "6|3|50000|NULL", "7|1|2000|NULL", "7|3|-2000|NULL",
		"8|1|100000|NULL", "8|4|-100000|NULL", "9|1|-5000|NULL", "9|4|5000|NULL",
		"10|1|-4000|NULL", "10|6|4000|u1", // the space account of the account's currency
		"11|2|117000|NULL", "11|7|-117000|u2",
		"12|1|-1500|NULL", "13|1|-1400|NULL", "14|1|-90|NULL")
}

func TestLinesCarryCategoryCurrencyAndItemNames(t *testing.T) {
	d, _, _ := converted(t)
	expect(t, d,
		`SELECT transaction_id, seq, category_id, currency_id, amount, note FROM transaction_lines ORDER BY 1, 2`,
		"1|0|1|1|1250|- Milk\n- Bread", // the items' names, trimmed, in item order, as a markdown list
		"2|0|2|1|300000|NULL", "12|0|1|1|1500|NULL", "13|0|1|1|1400|NULL",
		"14|0|1|1|90|NULL") // a blank item name leaves no note; amount 0 (tx 5) has no line
}

func TestDebtsMoveToTheirOwnTable(t *testing.T) {
	d, _, _ := converted(t)
	expect(t, d, `SELECT id, name, type, currency_id, initial_balance FROM accounts ORDER BY id`,
		"1|Card|bank|1|10000", "2|Cash|cash|2|0",
		"3|Loan to Sam|debt|1|0", "4|Mortgage|debt|1|0",
		"5|Old IOU|debt|1|3000", // no issuance: it starts from the principal
		"6|Other spaces · EUR|space|1|0", "7|Other spaces · RSD|space|2|0")
	expect(t, d, `SELECT account_id, direction, principal, due_date, counterparty, note, closed_at FROM debts ORDER BY 1`,
		"3|owed_to_me|50000|2027-01-01|Sam|for the car|NULL", "4|i_owe|100000|NULL|Bank|NULL|NULL",
		"5|owed_to_me|3000|NULL|Ann|NULL|2026-02-01T00:00:00Z")
	// What is still owed is the old target minus what was paid.
	expect(t, d, `SELECT a.id, a.initial_balance + SUM(p.amount) FROM accounts a JOIN payments p ON p.account_id = a.id
		WHERE a.id IN (3, 4) GROUP BY a.id ORDER BY a.id`, "3|48000", "4|-95000")
}

func TestTransfersKeepTheSignedBytes(t *testing.T) {
	d, _, _ := converted(t)
	sideA := `"from":{"space":"sp-a","account":1,"amount":4000,"currency":"EUR","decimals":2},` +
		`"to":{"space":"sp-b","account":9,"amount":468000,"currency":"RSD","decimals":2},`
	sideB := `"from":{"space":"sp-b","account":9,"amount":468000,"currency":"RSD","decimals":2},` +
		`"to":{"space":"sp-a","account":2,"amount":117000,"currency":"RSD","decimals":2},`
	// encoding/json escapes <, > and &: the bytes must be what was signed.
	u1 := `{"uuid":"u1",` + sideA + `"date":"2026-02-10","description":"Rent ` + "\\u003c" + `share` + "\\u003e" + ` ` + "\\u0026" + ` co",` +
		`"created_by":1,"version":1,"deleted_at":null}`
	u2 := `{"uuid":"u2",` + sideB + `"date":"2026-02-11","description":null,"created_by":null,"version":1,"deleted_at":null}`
	u4 := `{"uuid":"u4","from":{"space":"sp-a","account":1,"amount":700,"currency":"EUR","decimals":2},` +
		`"to":{"space":"sp-b","account":9,"amount":700,"currency":"EUR","decimals":2},` +
		`"date":"2026-02-13","description":"Fee","created_by":2,"version":3,"deleted_at":null}`
	u3 := `{"uuid":"u3","from":{"space":"sp-b","account":9,"amount":100,"currency":"EUR","decimals":2},` +
		`"to":{"space":"sp-a","account":1,"amount":100,"currency":"EUR","decimals":2},` +
		`"date":"2026-02-12","description":null,"created_by":null,"version":2,"deleted_at":null,` +
		`"sig_v":1,"signer_kid":"k2","signature":"sig3"}`
	expect(t, d, `SELECT uuid, payload, sig_v, source_kid, source_sig, status, remote FROM space_transfers ORDER BY uuid`,
		"u1|"+u1+"|1|k1|sig1|confirmed|NULL",
		"u2|"+u2+"|1|k1|sig2|confirmed|NULL", // needs_attention is not stored any more
		"u3|NULL|1|NULL|NULL|in_review|"+u3,  // only the other space has it: its record is the remote
		"u4|"+u4+`|1|k1|sig4|in_review|{"uuid":"u4","version":4}`)
	expect(t, d, `SELECT uuid, source, destination, version FROM space_transfers ORDER BY uuid`,
		"u1|sp-a|sp-b|1", "u2|sp-b|sp-a|1", "u3|sp-b|sp-a|2", "u4|sp-a|sp-b|3")
}

func TestRecurringRulesBecomeTemplateAndSchedule(t *testing.T) {
	d, _, _ := converted(t)
	expect(t, d, `SELECT id, template, schedule, is_active, error FROM recurring_transactions ORDER BY id`,
		`1|{"type":"expense","description":"Groceries","payments":[{"account_id":1,"amount":"-15","estimated":true}],`+
			`"lines":[{"category_id":1,"currency_id":1,"amount":"15","note":null}],"tag_ids":[1,2]}`+
			`|{"frequency":"monthly","interval":1,"day_of_month":1,"start":"2026-01-01"}|1|NULL`,
		// A transfer's receiving amount stays empty; with no occurrence the rule starts at its next run.
		`2|{"type":"transfer","description":null,"payments":[{"account_id":1,"amount":"-10"},`+
			`{"account_id":2,"amount":null}],"lines":[],"tag_ids":[]}`+
			`|{"frequency":"weekly","interval":2,"day_of_week":1,"start":"2026-02-02","end":"2026-12-31"}|0|NULL`,
		`3|{"type":"income","description":null,"payments":[{"account_id":2,"amount":null,"estimated":true}],`+
			`"lines":[],"tag_ids":[]}|{"frequency":"daily","interval":1,"start":"2026-01-01"}|1|NULL`)
	// The pending occurrence is due on the rule's next run, not on the date the transaction was moved to.
	expect(t, d, `SELECT recurring_transaction_id, due_date, transaction_id FROM recurring_instances ORDER BY 1, 2`,
		"1|2026-02-01|13", "1|2026-03-01|12")
}

// A stored template and schedule are read by the ledger and by the recurring
// package; what the converter writes must pass their strict decoding.
func TestConvertedRulesDecodeStrictly(t *testing.T) {
	d, _, _ := converted(t)
	rows, err := d.Query(`SELECT id, template, schedule FROM recurring_transactions`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var template, schedule string
		if err := rows.Scan(&id, &template, &schedule); err != nil {
			t.Fatal(err)
		}
		dec := json.NewDecoder(bytes.NewReader([]byte(template)))
		dec.DisallowUnknownFields()
		var in ledger.TxInput
		if err := dec.Decode(&in); err != nil {
			t.Errorf("rule %d template: %v", id, err)
		}
		if _, err := recurring.ParseSchedule(schedule); err != nil {
			t.Errorf("rule %d schedule: %v", id, err)
		}
	}
}

func TestConvertRefusesAnExistingTarget(t *testing.T) {
	src := openV1(t)
	dst := filepath.Join(t.TempDir(), "new.sqlite")
	if err := v1.Convert(context.Background(), src, dst); err != nil {
		t.Fatal(err)
	}
	if err := v1.Convert(context.Background(), src, dst); err == nil {
		t.Fatal("the second conversion overwrote the first result")
	}
}

// balances reads account id -> balance in minor units from a query.
func balances(t *testing.T, d *sql.DB, query string) map[int64]int64 {
	t.Helper()
	rows, err := d.Query(query)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	defer rows.Close()
	out := map[int64]int64{}
	for rows.Next() {
		var id, balance int64
		if err := rows.Scan(&id, &balance); err != nil {
			t.Fatal(err)
		}
		out[id] = balance
	}
	return out
}

// What a user sees must not move: every ordinary account and every debt has
// the balance it had in v1.0. v1.0 counted confirmed rows only, took a
// transfer's destination as to_amount (the amount when it is NULL, which the
// converter fixes), and a debt's balance was its target less what was paid.
func TestConvertKeepsEveryBalance(t *testing.T) {
	d, src, _ := converted(t)
	old := src2(t, src)
	const moved = `SELECT id, SUM(delta) FROM (
		SELECT account_id AS id, CASE WHEN type IN ('income', 'transfer_in', 'debt_borrow', 'debt_collection') THEN amount ELSE -amount END AS delta
			FROM transactions WHERE status = 'confirmed'
		UNION ALL
		SELECT to_account_id, COALESCE(to_amount, amount) FROM transactions
			WHERE status = 'confirmed' AND type = 'transfer' AND to_account_id IS NOT NULL
	) GROUP BY id`
	want := balances(t, old, `SELECT id, initial_balance FROM accounts WHERE type <> 'debt'`)
	for id, delta := range balances(t, old, moved) {
		if _, ok := want[id]; ok {
			want[id] += delta
		}
	}
	want2 := balances(t, old, `SELECT a.id, (CASE a.debt_type WHEN 'i_owe' THEN -1 ELSE 1 END) * (a.target_amount - COALESCE((
		SELECT SUM(COALESCE(t.to_amount, t.amount)) FROM transactions t WHERE t.to_account_id = a.id AND t.status = 'confirmed'
			AND t.type IN ('debt_payment', 'debt_collection')), 0)) FROM accounts a WHERE a.type = 'debt'`)
	for id, v := range want2 {
		want[id] = v
	}
	got := balances(t, d, `SELECT a.id, a.initial_balance + COALESCE((SELECT SUM(p.amount) FROM payments p
		JOIN transactions t ON t.id = p.transaction_id WHERE p.account_id = a.id AND t.status = 'confirmed'), 0)
		FROM accounts a WHERE a.type <> 'space'`)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("balances after the conversion\n got: %v\nwant: %v", got, want)
	}
}
