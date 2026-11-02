package ledger

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db"
	"savvy-go/internal/db/filter"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/money"
)

// Tx is a transaction with what it moved and what it was for: its payments
// (signed, each in its account's currency), its lines (always positive, each
// in a currency of its own) and its tags. Nothing ties payments to lines; the
// ledger compares them.
type Tx struct {
	ID                int64
	Type, Status      string
	Description, Date *string
	DayOrder          *int64
	ExternalID        *string
	CreatedBy         *int64
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
	Payments          []Payment
	Lines             []Line
	Tags              []Tag
	// TransferStatus is the status of the space transfer a payment belongs to
	// ("" when none); RecurringID the rule whose occurrence this is.
	TransferStatus string
	RecurringID    *int64
}

// Payment is money that moved on one account; negative when it left.
// SpaceTransferUUID is set only on a payment on a space account.
type Payment struct {
	AccountID         int64
	AccountType       string
	Amount            money.Money
	SpaceTransferUUID *string
}

// Line is an amount for a category, in the currency of Amount. Seq is its
// place within the transaction, from 0. A nil category is uncategorised.
type Line struct {
	Seq        int
	CategoryID *int64
	Amount     money.Money
	Note       *string
}

func txOf(x sqlc.GetTransactionRow) Tx {
	return Tx{
		ID: x.ID, Type: x.Type, Status: x.Status, Description: optString(x.Description), Date: optString(x.Date),
		DayOrder: optInt64(x.DayOrder), ExternalID: optString(x.ExternalID), CreatedBy: optInt64(x.CreatedBy),
		CreatedAt: optTime(x.CreatedAt), UpdatedAt: optTime(x.UpdatedAt),
		Payments: []Payment{}, Lines: []Line{}, Tags: []Tag{},
	}
}

// Transaction returns nil for an unknown id.
func (r Repo) Transaction(ctx context.Context, id int64) (*Tx, error) {
	x, err := r.q().GetTransaction(ctx, id)
	if err != nil {
		return nilIfMissing[Tx](err)
	}
	list := []Tx{txOf(x)}
	if err := r.load(ctx, list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

// TransferUUID is the uuid of the transfer between spaces a transaction is one
// side of (set on its payment on a space account); "" for an ordinary one.
func (r Repo) TransferUUID(ctx context.Context, id int64) string {
	tx, err := r.Transaction(ctx, id)
	if err != nil || tx == nil {
		return ""
	}
	for _, p := range tx.Payments {
		if p.SpaceTransferUUID != nil {
			return *p.SpaceTransferUUID
		}
	}
	return ""
}

// Filtered lists the transactions matching f, one page, and how many match in
// all. The payments, lines and tags of the page are read in three queries.
func (r Repo) Filtered(ctx context.Context, f filter.TxFilter, limit, offset int) ([]Tx, int, error) {
	total, err := filter.CountTransactions(ctx, r.DB, f)
	if err != nil {
		return nil, 0, err
	}
	rows, err := filter.ListTransactions(ctx, r.DB, f, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	out := make([]Tx, len(rows))
	for i, x := range rows {
		out[i] = txOf(x)
	}
	if err := r.load(ctx, out); err != nil {
		return nil, 0, err
	}
	return out, int(total), nil
}

// load fills the payments, lines and tags of txs: three queries for all.
func (r Repo) load(ctx context.Context, txs []Tx) error {
	if len(txs) == 0 {
		return nil
	}
	ids := make([]int64, len(txs))
	at := make(map[int64]int, len(txs))
	for i, t := range txs {
		ids[i] = t.ID
		at[t.ID] = i
	}
	payments, err := r.q().ListPaymentsOfTransactions(ctx, ids)
	if err != nil {
		return err
	}
	for _, p := range payments {
		t := &txs[at[p.TransactionID]]
		t.Payments = append(t.Payments, Payment{
			AccountID:         p.AccountID,
			AccountType:       p.AccountType,
			Amount:            money.New(p.Amount, money.Unit{ID: p.CurrencyID, Decimals: int(p.Decimals)}),
			SpaceTransferUUID: optString(p.SpaceTransferUuid),
		})
	}
	lines, err := r.q().ListLinesOfTransactions(ctx, ids)
	if err != nil {
		return err
	}
	for _, l := range lines {
		t := &txs[at[l.TransactionID]]
		t.Lines = append(t.Lines, Line{
			Seq: int(l.Seq), CategoryID: optInt64(l.CategoryID), Note: optString(l.Note),
			Amount: money.New(l.Amount, money.Unit{ID: l.CurrencyID, Decimals: int(l.Decimals)}),
		})
	}
	links, err := r.q().ListTransactionLinks(ctx, ids)
	if err != nil {
		return err
	}
	for _, l := range links {
		t := &txs[at[l.TransactionID]]
		t.TransferStatus = l.TransferStatus
		if l.RecurringID != 0 {
			id := l.RecurringID
			t.RecurringID = &id
		}
	}
	tags, err := r.TagsOf(ctx, ids)
	if err != nil {
		return err
	}
	for _, g := range tags {
		t := &txs[at[g.TransactionID]]
		t.Tags = append(t.Tags, g.Tag)
	}
	return nil
}

// TxFields is the writable part of a transaction row. ExternalID and
// CreatedBy are written on insert only; Status is changed by
// ConfirmTransaction and SkipTransaction.
type TxFields struct {
	Type, Status      string
	Description, Date *string
	ExternalID        *string
	CreatedBy         *int64
}

// InsertTransaction adds the row of a transaction (a blank Status is
// confirmed); its payments, lines and tags are stored by the methods below.
func (r Repo) InsertTransaction(ctx context.Context, t TxFields, now time.Time) (int64, error) {
	status := t.Status
	if status == "" {
		status = "confirmed"
	}
	res, err := r.q().InsertTransaction(ctx, sqlc.InsertTransactionParams{
		Type: t.Type, Description: db.NullString(t.Description), Date: db.NullString(t.Date), Status: status,
		ExternalID: db.NullString(t.ExternalID), CreatedBy: db.NullInt64(t.CreatedBy),
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateTransaction rewrites the type, description and date of a transaction.
// A move to another day drops its manual place in the day.
func (r Repo) UpdateTransaction(ctx context.Context, id int64, t TxFields, now time.Time) error {
	return r.q().UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		Type: t.Type, Description: db.NullString(t.Description), Date: db.NullString(t.Date),
		Column4: dateOrEmpty(t.Date), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

// DeleteTransaction deletes a transaction with its payments, lines and tags.
func (r Repo) DeleteTransaction(ctx context.Context, id int64) error {
	return r.q().DeleteTransaction(ctx, id)
}

// ConfirmTransaction settles a pending transaction on a date; its real
// amounts are written with ReplacePayments and ReplaceLines.
func (r Repo) ConfirmTransaction(ctx context.Context, id int64, date string, now time.Time) error {
	return r.q().ConfirmTransaction(ctx, sqlc.ConfirmTransactionParams{
		Date: db.NS(date), Column2: date, UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) SkipTransaction(ctx context.Context, id int64, now time.Time) error {
	return r.q().SkipTransaction(ctx, sqlc.SkipTransactionParams{UpdatedAt: db.TimeNS(now), ID: id})
}

// HasExternalID reports whether a transaction with this external id exists.
func (r Repo) HasExternalID(ctx context.Context, externalID string) (bool, error) {
	exists, err := r.q().ExistsTransactionExternalID(ctx, db.NS(externalID))
	return exists, err
}

// PaymentFields is a payment to store. SpaceTransferUUID is written when the
// payment is inserted, never when it is updated.
type PaymentFields struct {
	AccountID         int64
	Amount            money.Money
	SpaceTransferUUID *string
}

// ReplacePayments makes the payments of a transaction the given ones. A
// payment whose account is already there is updated in place, so its
// SpaceTransferUUID stays; one whose account is not in the list is deleted.
func (r Repo) ReplacePayments(ctx context.Context, txID int64, payments []PaymentFields) error {
	have, err := r.q().ListPaymentAccountIDs(ctx, txID)
	if err != nil {
		return err
	}
	had := make(map[int64]bool, len(have))
	for _, id := range have {
		had[id] = true
	}
	keep := make(map[int64]bool, len(payments))
	for _, p := range payments {
		keep[p.AccountID] = true
		if had[p.AccountID] {
			err = r.q().UpdatePaymentAmount(ctx, sqlc.UpdatePaymentAmountParams{
				Amount: p.Amount.Minor(), TransactionID: txID, AccountID: p.AccountID,
			})
		} else {
			err = r.q().InsertPayment(ctx, sqlc.InsertPaymentParams{
				TransactionID: txID, AccountID: p.AccountID, Amount: p.Amount.Minor(),
				SpaceTransferUuid: db.NullString(p.SpaceTransferUUID),
			})
		}
		if err != nil {
			return err
		}
	}
	for _, id := range have {
		if keep[id] {
			continue
		}
		if err := r.q().DeletePayment(ctx, sqlc.DeletePaymentParams{TransactionID: txID, AccountID: id}); err != nil {
			return err
		}
	}
	return nil
}

// LineFields is a line to store; its currency is the one of Amount.
type LineFields struct {
	CategoryID *int64
	Amount     money.Money
	Note       *string
}

// ReplaceLines rewrites the lines of a transaction, numbered from 0 in the
// order given.
func (r Repo) ReplaceLines(ctx context.Context, txID int64, lines []LineFields) error {
	if err := r.q().DeleteTransactionLines(ctx, txID); err != nil {
		return err
	}
	for i, l := range lines {
		if err := r.q().InsertTransactionLine(ctx, sqlc.InsertTransactionLineParams{
			TransactionID: txID, Seq: int64(i), CategoryID: db.NullInt64(l.CategoryID),
			CurrencyID: l.Amount.Unit().ID, Amount: l.Amount.Minor(), Note: db.NullString(l.Note),
		}); err != nil {
			return err
		}
	}
	return nil
}

// TxTag is a tag with the transaction it is on.
type TxTag struct {
	TransactionID int64
	Tag
}

func (r Repo) TagsOf(ctx context.Context, txIDs []int64) ([]TxTag, error) {
	rows, err := r.q().ListTagsOfTransactions(ctx, txIDs)
	if err != nil {
		return nil, err
	}
	out := make([]TxTag, len(rows))
	for i, x := range rows {
		out[i] = TxTag{TransactionID: x.TransactionID, Tag: tagOf(x.ID, x.Name, x.CreatedAt, 0)}
	}
	return out, nil
}

// SetTags replaces the tags of a transaction.
func (r Repo) SetTags(ctx context.Context, txID int64, tagIDs []int64) error {
	if err := r.q().DeleteTransactionTags(ctx, txID); err != nil {
		return err
	}
	for _, id := range tagIDs {
		if err := r.q().InsertTransactionTag(ctx,
			sqlc.InsertTransactionTagParams{TransactionID: txID, TagID: id}); err != nil {
			return err
		}
	}
	return nil
}

// SummaryRow is one line's amount with what is needed to fold it into the
// base currency.
type SummaryRow struct {
	Type     string
	Minor    int64
	Decimals int
	Rate     decimal.Decimal
	IsBase   bool
}

// SummaryRows lists the lines of the income and expense in a status.
func (r Repo) SummaryRows(ctx context.Context, status string) ([]SummaryRow, error) {
	rows, err := r.q().ListTransactionSummaryRows(ctx, status)
	if err != nil {
		return nil, err
	}
	out := make([]SummaryRow, len(rows))
	for i, x := range rows {
		out[i] = SummaryRow{Type: x.Type, Minor: x.Amount, Decimals: int(x.Decimals), Rate: x.Rate, IsBase: x.IsBase != 0}
	}
	return out, nil
}

// BudgetSpent is what the lines of the expenses in [start, end] of the given
// categories and tags spent, in unit, converted at targetRate.
func (r Repo) BudgetSpent(
	ctx context.Context,
	start, end string,
	categoryIDs, tagIDs []int64,
	unit money.Unit,
	targetRate decimal.Decimal,
) (money.Money, error) {
	return filter.BudgetSpent(ctx, r.DB, start, end, categoryIDs, tagIDs, unit, targetRate)
}

// SetTransactionCategory gives a category to the lines of a transaction that
// have none.
func (r Repo) SetTransactionCategory(ctx context.Context, id, categoryID int64, now time.Time) error {
	if err := r.q().SetTransactionLinesCategory(ctx,
		sqlc.SetTransactionLinesCategoryParams{CategoryID: db.NI(categoryID), TransactionID: id}); err != nil {
		return err
	}
	return r.q().TouchTransaction(ctx, sqlc.TouchTransactionParams{UpdatedAt: db.TimeNS(now), ID: id})
}

// SetTransactionDescription changes only a transaction's description.
func (r Repo) SetTransactionDescription(ctx context.Context, id int64, description string, now time.Time) error {
	return r.q().
		UpdateTransactionDescription(ctx,
			sqlc.UpdateTransactionDescriptionParams{Description: db.NS(description), UpdatedAt: db.TimeNS(now), ID: id})
}

// CountTransactions counts every transaction of the space.
func (r Repo) CountTransactions(ctx context.Context) int {
	n, _ := r.q().CountAllTransactions(ctx)
	return int(n)
}

// CountAccounts counts every account of the space.
func (r Repo) CountAccounts(ctx context.Context) int {
	n, _ := r.q().CountAccounts(ctx)
	return int(n)
}

// DaySlot is where a transaction sits in the list: its calendar day and the
// key the list sorts that day by.
type DaySlot struct {
	ID  int64
	Day string
	Key int64
}

// DaySlots returns the slots of the given ids; an unknown id has none.
func (r Repo) DaySlots(ctx context.Context, ids []int64) ([]DaySlot, error) {
	rows, err := r.q().ListTransactionDayOrder(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]DaySlot, len(rows))
	for i, x := range rows {
		out[i] = DaySlot{ID: x.ID, Day: x.Day, Key: x.SortKey}
	}
	return out, nil
}

// SetDayOrder places a transaction at a sort key within its day.
func (r Repo) SetDayOrder(ctx context.Context, id, key int64) error {
	return r.q().SetTransactionDayOrder(ctx, sqlc.SetTransactionDayOrderParams{DayOrder: db.NI(key), ID: id})
}

func dateOrEmpty(d *string) string {
	if d == nil {
		return ""
	}
	return *d
}
