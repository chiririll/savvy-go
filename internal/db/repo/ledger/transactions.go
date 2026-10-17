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

// Tx is a transaction row. Amount is in the currency of its account and
// ToAmount in the currency of the to-account.
type Tx struct {
	ID                      int64
	Type, Status            string
	AccountID               int64
	ToAccountID, CategoryID *int64
	Amount                  money.Money
	ToAmount                *money.Money
	IsEstimated             bool
	Description, Date       *string
	RecurringID             *int64
	CreatedAt               *time.Time
}

func txOf(x sqlc.GetTransactionRow) Tx {
	return Tx{
		ID: x.ID, Type: x.Type, Status: x.Status, AccountID: x.AccountID,
		ToAccountID: optInt64(x.ToAccountID), CategoryID: optInt64(x.CategoryID),
		Amount:      money.New(x.Amount, money.Unit{ID: x.CurrencyID, Decimals: int(x.Decimals)}),
		ToAmount:    MoneyFromNull(x.ToAmount, money.Unit{ID: x.ToCurrencyID, Decimals: int(x.ToDecimals)}),
		IsEstimated: x.IsEstimated != 0, Description: optString(x.Description), Date: optString(x.Date),
		RecurringID: optInt64(x.RecurringTransactionID), CreatedAt: optTime(x.CreatedAt),
	}
}

// Transaction returns nil for an unknown id.
func (r Repo) Transaction(ctx context.Context, id int64) (*Tx, error) {
	x, err := r.q().GetTransaction(ctx, id)
	if err != nil {
		return nilIfMissing[Tx](err)
	}
	t := txOf(x)
	return &t, nil
}

// TransferUUID is the uuid of the transfer between spaces this transaction is
// one side of; "" when it is an ordinary transaction.
func (r Repo) TransferUUID(ctx context.Context, id int64) string {
	v, err := r.q().GetTransactionTransferUUID(ctx, id)
	if err != nil {
		return ""
	}
	return v.String
}

// Filtered lists the transactions matching f, newest first, one page, and
// how many match in all.
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
	return out, int(total), nil
}

// TxFields is the writable part of a transaction; amounts are already in the
// currencies of the accounts.
type TxFields struct {
	Type, Status            string
	AccountID               int64
	ToAccountID, CategoryID *int64
	Amount                  money.Money
	ToAmount                *money.Money
	IsEstimated             bool
	Description, Date       *string
	RecurringID             *int64 // insert only
}

func (r Repo) InsertTransaction(ctx context.Context, t TxFields, now time.Time) (int64, error) {
	res, err := r.q().InsertTransaction(ctx, sqlc.InsertTransactionParams{
		Type: t.Type, AccountID: t.AccountID, ToAccountID: db.NullInt64(t.ToAccountID),
		CategoryID: db.NullInt64(t.CategoryID), Amount: t.Amount.Minor(), ToAmount: db.NullMoney(t.ToAmount),
		IsEstimated: db.BoolInt(t.IsEstimated), Description: db.NullString(t.Description),
		Date: db.NullString(t.Date), Status: t.Status, RecurringTransactionID: db.NullInt64(t.RecurringID),
		CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
	})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateTransaction rewrites a transaction's fields; Status and RecurringID
// are not touched.
func (r Repo) UpdateTransaction(ctx context.Context, id int64, t TxFields, now time.Time) error {
	return r.q().UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		Type: t.Type, AccountID: t.AccountID, ToAccountID: db.NullInt64(t.ToAccountID),
		CategoryID: db.NullInt64(t.CategoryID), Amount: t.Amount.Minor(), ToAmount: db.NullMoney(t.ToAmount),
		IsEstimated: db.BoolInt(t.IsEstimated), Description: db.NullString(t.Description),
		Date: db.NullString(t.Date), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) DeleteTransaction(ctx context.Context, id int64) error {
	return r.q().DeleteTransaction(ctx, id)
}

// ConfirmTransaction settles a pending transaction with its real amounts.
func (r Repo) ConfirmTransaction(
	ctx context.Context,
	id int64,
	amount money.Money,
	toAmount *money.Money,
	date string,
	now time.Time,
) error {
	return r.q().ConfirmTransaction(ctx, sqlc.ConfirmTransactionParams{
		Amount: amount.Minor(), ToAmount: db.NullMoney(toAmount), Date: db.NS(date), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) SkipTransaction(ctx context.Context, id int64, now time.Time) error {
	return r.q().SkipTransaction(ctx, sqlc.SkipTransactionParams{UpdatedAt: db.TimeNS(now), ID: id})
}

// Items and tags of transactions

// TxItem is a line of a transaction, in minor units of its account's currency.
type TxItem struct {
	ID, TransactionID int64
	Name              string
	Quantity          decimal.Decimal
	PricePerUnit      int64
	TotalPrice        int64
}

// NewTxItem is a line to store.
type NewTxItem struct {
	Name         string
	Quantity     decimal.Decimal
	PricePerUnit money.Money
	TotalPrice   money.Money
}

func (r Repo) ItemsOf(ctx context.Context, txIDs []int64) ([]TxItem, error) {
	rows, err := r.q().ListItemsOfTransactions(ctx, txIDs)
	if err != nil {
		return nil, err
	}
	out := make([]TxItem, len(rows))
	for i, x := range rows {
		out[i] = TxItem{
			ID:            x.ID,
			TransactionID: x.TransactionID,
			Name:          x.Name,
			Quantity:      x.Quantity,
			PricePerUnit:  x.PricePerUnit,
			TotalPrice:    x.TotalPrice,
		}
	}
	return out, nil
}

func (r Repo) InsertItems(ctx context.Context, txID int64, items []NewTxItem, now time.Time) error {
	for _, it := range items {
		if err := r.q().InsertTransactionItem(ctx, sqlc.InsertTransactionItemParams{
			TransactionID: txID, Name: it.Name, Quantity: it.Quantity,
			PricePerUnit: it.PricePerUnit.Minor(), TotalPrice: it.TotalPrice.Minor(),
			CreatedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r Repo) DeleteItems(ctx context.Context, txID int64) error {
	return r.q().DeleteTransactionItems(ctx, txID)
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
	_ = r.q().DeleteTransactionTags(ctx, txID)
	for _, id := range tagIDs {
		if err := r.q().InsertTransactionTag(ctx,
			sqlc.InsertTransactionTagParams{TransactionID: txID, TagID: id}); err != nil {
			return err
		}
	}
	return nil
}

// SummaryRow is one transaction's amount with what is needed to fold it into
// the base currency.
type SummaryRow struct {
	Type     string
	Minor    int64
	Decimals int
	Rate     decimal.Decimal
	IsBase   bool
}

// SummaryRows lists the amounts of the transactions in a status.
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

// BudgetSpent is what the transactions in [start, end] of the given
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

// SetTransactionCategory changes only a transaction's category.
func (r Repo) SetTransactionCategory(ctx context.Context, id, categoryID int64, now time.Time) error {
	return r.q().
		UpdateTransactionCategory(ctx,
			sqlc.UpdateTransactionCategoryParams{CategoryID: db.NI(categoryID), UpdatedAt: db.TimeNS(now), ID: id})
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
