package v1

import (
	"context"
	"database/sql"

	"github.com/shopspring/decimal"
)

// The template and the schedule are frozen copies of the JSON of
// ledger.TxInput and recurring.Schedule as v1.1 wrote them first; this package
// sits below both and cannot import them. The converter test decodes the
// result with the real types.
type templateV1 struct {
	Type        string      `json:"type"`
	Description *string     `json:"description"`
	Payments    []paymentV1 `json:"payments"`
	Lines       []lineV1    `json:"lines"`
	TagIDs      []int64     `json:"tag_ids"`
}

type paymentV1 struct {
	AccountID int64   `json:"account_id"`
	Amount    *string `json:"amount"`
	Estimated bool    `json:"estimated,omitempty"`
}

type lineV1 struct {
	CategoryID *int64  `json:"category_id"`
	CurrencyID int64   `json:"currency_id"`
	Amount     string  `json:"amount"`
	Note       *string `json:"note"`
}

type scheduleV1 struct {
	Frequency  string  `json:"frequency"`
	Interval   int     `json:"interval"`
	DayOfWeek  *int    `json:"day_of_week,omitempty"`
	DayOfMonth *int    `json:"day_of_month,omitempty"`
	Start      string  `json:"start"`
	End        *string `json:"end,omitempty"`
}

// oldRule is a v1.0 recurring_transactions row with what the template needs
// of its accounts.
type oldRule struct {
	id, accountID           int64
	kind                    string
	toAccountID, categoryID *int64
	amount                  int64
	toAmount                *int64
	estimated               bool
	description             *string
	frequency               string
	interval                int
	dayOfWeek, dayOfMonth   *int
	start, nextRun          string
	end                     *string
	active                  bool
	createdAt, updatedAt    *string
	currencyID              int64
	decimals                int32
	toDecimals              *int32
	hasInstance             bool
	tags                    []int64
}

const readRules = `SELECT r.id, r.type, r.account_id, r.to_account_id, r.category_id, r.amount, r.to_amount,
	r.is_estimated, r.description, r.frequency, r.interval, r.day_of_week, r.day_of_month, r.start_date,
	r.end_date, r.next_run_date, r.is_active, r.created_at, r.updated_at, a.currency_id, c.decimals, tc.decimals,
	EXISTS (SELECT 1 FROM main.recurring_instances i WHERE i.recurring_transaction_id = r.id)
FROM src.recurring_transactions r
JOIN src.accounts a ON a.id = r.account_id
JOIN main.currencies c ON c.id = a.currency_id
LEFT JOIN src.accounts ta ON ta.id = r.to_account_id
LEFT JOIN main.currencies tc ON tc.id = ta.currency_id
ORDER BY r.id`

func loadRules(ctx context.Context, tx *sql.Tx) ([]oldRule, error) {
	rows, err := tx.QueryContext(ctx, readRules)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []oldRule
	for rows.Next() {
		var r oldRule
		if err := rows.Scan(&r.id, &r.kind, &r.accountID, &r.toAccountID, &r.categoryID, &r.amount, &r.toAmount,
			&r.estimated, &r.description, &r.frequency, &r.interval, &r.dayOfWeek, &r.dayOfMonth, &r.start,
			&r.end, &r.nextRun, &r.active, &r.createdAt, &r.updatedAt, &r.currencyID, &r.decimals, &r.toDecimals,
			&r.hasInstance); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func loadRuleTags(ctx context.Context, tx *sql.Tx) (map[int64][]int64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT recurring_transaction_id, tag_id
		FROM src.recurring_transaction_tag ORDER BY 1, 2`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := map[int64][]int64{}
	for rows.Next() {
		var rule, tag int64
		if err := rows.Scan(&rule, &tag); err != nil {
			return nil, err
		}
		tags[rule] = append(tags[rule], tag)
	}
	return tags, rows.Err()
}

// copyRules turns each rule into a template and a schedule. The instances of
// the rules were written with the transactions.
func copyRules(ctx context.Context, tx *sql.Tx) error {
	rules, err := loadRules(ctx, tx)
	if err != nil {
		return err
	}
	tags, err := loadRuleTags(ctx, tx)
	if err != nil {
		return err
	}
	for _, r := range rules {
		r.tags = tags[r.id]
		if _, err := tx.ExecContext(ctx, `INSERT INTO main.recurring_transactions
			(id, template, schedule, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			r.id, *marshal(r.template()), *marshal(r.schedule()), r.active, r.createdAt, r.updatedAt); err != nil {
			return err
		}
	}
	return bumpSequence(ctx, tx, "recurring_transactions")
}

// money is a minor amount as the decimal the API reads, in major units.
func money(minor int64, decimals int32) *string {
	s := decimal.New(minor, -decimals).String()
	return &s
}

// template is the transaction the rule creates, entered as for the API. An
// amount of 0 stays empty: a payment of 0 is refused on new input, and an
// empty one is filled when an occurrence is made. A transfer's receiving
// amount stays empty when v1.0 had none.
func (r oldRule) template() templateV1 {
	t := templateV1{Type: r.kind, Description: r.description, Lines: []lineV1{}, TagIDs: []int64{}}
	if r.tags != nil {
		t.TagIDs = r.tags
	}
	sign := int64(-1) // expense and transfer leave the account
	if r.kind == "income" {
		sign = 1
	}
	first := paymentV1{AccountID: r.accountID, Estimated: r.estimated}
	if r.amount != 0 {
		first.Amount = money(sign*r.amount, r.decimals)
	}
	t.Payments = []paymentV1{first}
	switch {
	case r.kind == "transfer" && r.toAccountID != nil:
		second := paymentV1{AccountID: *r.toAccountID}
		if r.toAmount != nil && r.toDecimals != nil {
			second.Amount = money(*r.toAmount, *r.toDecimals)
		}
		t.Payments = append(t.Payments, second)
	case r.kind != "transfer" && r.amount != 0:
		t.Lines = []lineV1{{CategoryID: r.categoryID, CurrencyID: r.currencyID, Amount: *money(r.amount, r.decimals)}}
	}
	return t
}

// schedule is the old schedule columns. A day that does not fit the frequency
// is dropped (v1.0 did not stop it being stored). A rule with no handled due
// date starts at its next run when that is later, so the date derived from
// the schedule is the one the rule had.
func (r oldRule) schedule() scheduleV1 {
	s := scheduleV1{Frequency: r.frequency, Interval: r.interval, Start: r.start, End: r.end}
	if r.frequency == "weekly" {
		s.DayOfWeek = r.dayOfWeek
	}
	if r.frequency == "monthly" {
		s.DayOfMonth = r.dayOfMonth
	}
	if !r.hasInstance && r.nextRun > r.start {
		s.Start = r.nextRun
	}
	return s
}
