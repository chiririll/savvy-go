package ledger

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/db/filter"
	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

// appLocation is the timezone used for calendar-day concepts across this
// package: "today", and whether a date is in the future. Configured once at
// startup from the TZ env var (see app.Config.Location); defaults to UTC.
// A raw time.Now().UTC() day boundary would disagree with a client whose
// local day has already rolled over, wrongly flagging same-day dates as future.
var appLocation = time.UTC

// Today is the current calendar day, "YYYY-MM-DD", in the app timezone.
func Today() string { return time.Now().In(appLocation).Format("2006-01-02") }

// SetLocation configures appLocation. Call once during server startup.
func SetLocation(loc *time.Location) {
	if loc != nil {
		appLocation = loc
	}
}

// TxItem is a line of a transaction, priced in the currency of its account.
type TxItem struct {
	ID           int64
	Name         string
	Quantity     decimal.Decimal
	PricePerUnit money.Money
	TotalPrice   money.Money
}

// TxItemInput is a line as entered. TotalPrice is Quantity times PricePerUnit
// when zero.
type TxItemInput struct {
	Name         string
	Quantity     decimal.Decimal
	PricePerUnit decimal.Decimal
	TotalPrice   decimal.Decimal
}

type Transaction struct {
	ID          int64
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      money.Money  // in the currency of Account
	ToAmount    *money.Money // in the currency of ToAccount
	IsEstimated bool         // the amount is a rough guess until the transaction is confirmed
	Description *string
	Date        *string
	Status      string
	RecurringID *int64
	CreatedAt   *time.Time
	GroupUUID   *string // the payment this transaction is a leg of, see Groups
	Account     *Account
	ToAccount   *Account
	Category    *Category
	Items       []TxItem
	Tags        []Tag
}

type TxInput struct {
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      decimal.Decimal
	ToAmount    *decimal.Decimal
	IsEstimated bool // only kept for a pending transaction, see keepsEstimate
	Description *string
	Date        *string
	Status      *string
	RecurringID *int64
	TagIDs      []int64
	Items       []TxItemInput

	group *string // set by Groups only
}

type Transactions struct{ DB store.DB }

func (s Transactions) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// ErrTransferRow is returned for a change to one side of a transfer between
// spaces: the transfer changes, not its rows.
var ErrTransferRow = errors.New(
	"this transaction is one side of a transfer between spaces; change the transfer instead",
)

// transferSideType reports the types reserved for a transfer between spaces;
// they are written only by Transfers.
func transferSideType(typ string) bool { return typ == "transfer_out" || typ == "transfer_in" }

// TxActions is what may be done to a transaction. The Transactions service
// refuses what is not allowed, and the API reports the same flags to clients.
type TxActions struct {
	Edit, Delete, Duplicate, Confirm, Skip bool
}

// Allowed is the single source of truth for TxActions. One side of a transfer
// between spaces changes only through the transfer, never as a transaction.
func (t Transaction) Allowed() TxActions {
	switch {
	case transferSideType(t.Type):
		return TxActions{}
	case t.GroupUUID != nil:
		// A leg changes with its payment, through Groups.
		return TxActions{}
	case t.Status == "skipped":
		return TxActions{Delete: true}
	case t.Status == "pending" && t.RecurringID != nil:
		// The open occurrence belongs to its recurrence, which replans it. Once
		// confirmed or skipped it is an ordinary transaction.
		return TxActions{Confirm: true, Skip: true}
	case t.Status == "pending":
		return TxActions{Edit: true, Delete: true, Duplicate: true, Confirm: true}
	default:
		return TxActions{Edit: true, Delete: true, Duplicate: true}
	}
}

func (s Transactions) Create(ctx context.Context, in TxInput) (*Transaction, error) {
	if transferSideType(in.Type) {
		return nil, ErrTransferRow
	}
	status := "confirmed"
	if in.Status != nil {
		status = *in.Status
	} else if in.Date == nil || IsFuture(*in.Date) {
		status = "pending"
	}
	if status == "confirmed" {
		if in.Date == nil {
			return nil, fmt.Errorf("date required")
		}
		if IsFuture(*in.Date) {
			return nil, fmt.Errorf("future")
		}
	}
	amount, toAmount, err := s.amounts(ctx, in)
	if err != nil {
		return nil, err
	}
	items, err := itemRows(in.Items, amount.Unit())
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	id, err := s.repo().InsertTransaction(ctx, repo.TxFields{
		Type: in.Type, Status: status, AccountID: in.AccountID, ToAccountID: in.ToAccountID, CategoryID: in.CategoryID,
		Amount: amount, ToAmount: toAmount, IsEstimated: in.IsEstimated && keepsEstimate(in.Type, status, len(items)),
		Description: in.Description, Date: in.Date, RecurringID: in.RecurringID, GroupUUID: in.group,
	}, now)
	if err != nil {
		return nil, err
	}
	if err := s.repo().InsertItems(ctx, id, items, now); err != nil {
		return nil, err
	}
	if err := s.SetTags(ctx, id, in.TagIDs); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Update(ctx context.Context, id int64, in TxInput) (*Transaction, error) {
	if err := s.guardRow(ctx, id); err != nil {
		return nil, err
	}
	if transferSideType(in.Type) {
		return nil, ErrTransferRow
	}
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if !cur.Allowed().Edit {
		return nil, fmt.Errorf("cannot edit")
	}
	amount, toAmount, err := s.amounts(ctx, in)
	if err != nil {
		return nil, err
	}
	items, err := itemRows(in.Items, amount.Unit())
	if err != nil {
		return nil, err
	}
	itemCount := len(cur.Items) // items not sent are kept
	if in.Items != nil {
		itemCount = len(items)
	}
	now := time.Now().UTC()
	err = s.repo().UpdateTransaction(ctx, id, repo.TxFields{
		Type: in.Type, AccountID: in.AccountID, ToAccountID: in.ToAccountID, CategoryID: in.CategoryID,
		Amount: amount, ToAmount: toAmount, IsEstimated: in.IsEstimated && keepsEstimate(in.Type, cur.Status, itemCount),
		Description: in.Description, Date: in.Date,
	}, now)
	if err != nil {
		return nil, err
	}
	if in.Items != nil {
		_ = s.repo().DeleteItems(ctx, id)
		if err := s.repo().InsertItems(ctx, id, items, now); err != nil {
			return nil, err
		}
	}
	if in.TagIDs != nil {
		if err := s.SetTags(ctx, id, in.TagIDs); err != nil {
			return nil, err
		}
	}
	return s.ByID(ctx, id)
}

// ErrReorder is returned for a reorder the list cannot show: ids that repeat,
// do not exist or are not on the day.
var ErrReorder = errors.New("reorder: ids must be distinct transactions of that day")

// Reorder puts the transactions ids, oldest first, in that order within
// the calendar day (YYYY-MM-DD). The ids may be only some of the day's
// transactions (a filtered list): they trade the places they already hold and
// the rest stay where they are. Across days nothing moves.
func (s Transactions) Reorder(ctx context.Context, day string, ids []int64) error {
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return fmt.Errorf("day must be YYYY-MM-DD")
	}
	if len(ids) == 0 {
		return nil
	}
	return store.Tx(ctx, s.DB, func(d store.DB) error {
		r := repo.Repo{DB: d}
		slots, err := r.DaySlots(ctx, ids)
		if err != nil {
			return err
		}
		if len(slots) != len(ids) {
			return ErrReorder
		}
		keys := make([]int64, 0, len(slots))
		seen := make(map[int64]bool, len(ids))
		for _, sl := range slots {
			if sl.Day != day || seen[sl.ID] {
				return ErrReorder
			}
			seen[sl.ID] = true
			keys = append(keys, sl.Key)
		}
		slices.Sort(keys)
		for i, id := range ids {
			if err := r.SetDayOrder(ctx, id, keys[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// Plan is what a recurrence plans for its next, still pending, occurrence.
type Plan struct {
	Type        string
	AccountID   int64
	ToAccountID *int64
	CategoryID  *int64
	Amount      money.Money
	ToAmount    *money.Money
	IsEstimated bool
	Description *string
	Date        string
	TagIDs      []int64
}

// Replan rewrites a pending occurrence to follow its recurrence, which owns
// its content (so it bypasses the edit rules of Update).
func (s Transactions) Replan(ctx context.Context, id int64, p Plan) error {
	date := p.Date
	err := s.repo().UpdateTransaction(ctx, id, repo.TxFields{
		Type: p.Type, AccountID: p.AccountID, ToAccountID: p.ToAccountID, CategoryID: p.CategoryID,
		Amount: p.Amount, ToAmount: p.ToAmount, IsEstimated: p.IsEstimated, Description: p.Description, Date: &date,
	}, time.Now().UTC())
	if err != nil {
		return err
	}
	return s.SetTags(ctx, id, p.TagIDs)
}

// guardRow refuses changing a transaction that mirrors a transfer between
// spaces (P31), or is a leg of a payment: such rows change only through the
// transfer or the group.
func (s Transactions) guardRow(ctx context.Context, id int64) error {
	if s.repo().TransferUUID(ctx, id) != "" {
		return ErrTransferRow
	}
	if s.repo().GroupUUID(ctx, id) != "" {
		return ErrGroupLeg
	}
	return nil
}

func (s Transactions) Delete(ctx context.Context, id int64) error {
	if err := s.guardRow(ctx, id); err != nil {
		return err
	}
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return err
	}
	if !cur.Allowed().Delete {
		return fmt.Errorf("cannot delete recurring")
	}
	return s.repo().DeleteTransaction(ctx, id)
}

// Confirm settles a pending transaction on date. The amounts it was planned
// with stand; only an estimate can be replaced by the real amounts, and a
// transfer given only amount delivers it converted at the current rates.
func (s Transactions) Confirm(
	ctx context.Context,
	id int64,
	date string,
	amount, toAmount *decimal.Decimal,
) (*Transaction, error) {
	if err := s.guardRow(ctx, id); err != nil {
		return nil, err
	}
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if !cur.Allowed().Confirm {
		return nil, fmt.Errorf("not pending")
	}
	if date == "" && cur.Date != nil {
		date = *cur.Date
	}
	if date == "" {
		return nil, fmt.Errorf("date required")
	}
	if IsFuture(date) {
		return nil, fmt.Errorf("future")
	}
	actual, actualTo := cur.Amount, cur.ToAmount
	if amount != nil || toAmount != nil {
		if !cur.IsEstimated {
			return nil, fmt.Errorf("not estimated")
		}
		in := TxInput{
			Type:        cur.Type,
			AccountID:   cur.AccountID,
			ToAccountID: cur.ToAccountID,
			Amount:      cur.Amount.Decimal(),
			ToAmount:    toAmount,
		}
		if amount != nil {
			in.Amount = *amount
		}
		if actual, actualTo, err = s.amounts(ctx, in); err != nil {
			return nil, err
		}
	}
	if err := s.repo().ConfirmTransaction(ctx, id, actual, actualTo, date, time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Skip(ctx context.Context, id int64) (*Transaction, error) {
	if err := s.guardRow(ctx, id); err != nil {
		return nil, err
	}
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if !cur.Allowed().Skip {
		return nil, fmt.Errorf("cannot skip")
	}
	if err := s.repo().SkipTransaction(ctx, id, time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Transactions) Duplicate(ctx context.Context, id int64) (*Transaction, error) {
	if err := s.guardRow(ctx, id); err != nil {
		return nil, err
	}
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if !cur.Allowed().Duplicate {
		return nil, fmt.Errorf("cannot duplicate")
	}
	today := time.Now().In(appLocation).Format("2006-01-02")
	in := TxInput{
		Type: cur.Type, AccountID: cur.AccountID, ToAccountID: cur.ToAccountID,
		CategoryID: cur.CategoryID, Amount: cur.Amount.Decimal(),
		Description: cur.Description, Date: &today,
	}
	if cur.ToAmount != nil {
		d := cur.ToAmount.Decimal()
		in.ToAmount = &d
	}
	for _, it := range cur.Items {
		in.Items = append(in.Items, TxItemInput{
			Name: it.Name, Quantity: it.Quantity,
			PricePerUnit: it.PricePerUnit.Decimal(), TotalPrice: it.TotalPrice.Decimal(),
		})
	}
	for _, t := range cur.Tags {
		in.TagIDs = append(in.TagIDs, t.ID)
	}
	st := "confirmed"
	in.Status = &st
	return s.Create(ctx, in)
}

func (s Transactions) ByID(ctx context.Context, id int64) (*Transaction, error) {
	r, err := s.repo().Transaction(ctx, id)
	if r == nil || err != nil {
		return nil, err
	}
	list, err := s.hydrate(ctx, []repo.Tx{*r})
	if err != nil {
		return nil, err
	}
	return &list[0], nil
}

func (s Transactions) Filtered(ctx context.Context, f filter.TxFilter, page, perPage int) ([]Transaction, int, error) {
	if perPage <= 0 {
		perPage = 25
	}
	if page <= 0 {
		page = 1
	}
	rows, total, err := s.repo().Filtered(ctx, f, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, err
	}
	list, err := s.hydrate(ctx, rows)
	return list, total, err
}

// SetCategory changes only a transaction's category (an automation action).
func (s Transactions) SetCategory(ctx context.Context, id, categoryID int64) error {
	return s.repo().SetTransactionCategory(ctx, id, categoryID, time.Now().UTC())
}

// SetDescription changes only a transaction's description (an automation action).
func (s Transactions) SetDescription(ctx context.Context, id int64, description string) error {
	return s.repo().SetTransactionDescription(ctx, id, description, time.Now().UTC())
}

// SetTags replaces the tags of a transaction.
func (s Transactions) SetTags(ctx context.Context, txID int64, ids []int64) error {
	return s.repo().SetTags(ctx, txID, ids)
}

// Units returns the unit of the account and, when set, of the to-account;
// the to-account falls back to the account's unit.
func (s Transactions) Units(ctx context.Context, accountID int64, toAccountID *int64) (money.Unit, money.Unit, error) {
	accts := Accounts{DB: s.DB}
	unit, err := accts.Unit(ctx, accountID)
	if err != nil {
		return money.Unit{}, money.Unit{}, err
	}
	if toAccountID == nil {
		return unit, unit, nil
	}
	toUnit, err := accts.Unit(ctx, *toAccountID)
	if err != nil {
		return money.Unit{}, money.Unit{}, err
	}
	return unit, toUnit, nil
}

// amounts turns the entered amounts into money in the currencies of the
// accounts. A transfer without a destination amount delivers the amount
// converted at the current rates, never the same number reinterpreted in
// another currency.
func (s Transactions) amounts(ctx context.Context, in TxInput) (amount money.Money, toAmount *money.Money, err error) {
	unit, toUnit, err := s.Units(ctx, in.AccountID, in.ToAccountID)
	if err != nil {
		return money.Money{}, nil, err
	}
	if amount, err = money.FromInput(in.Amount, unit); err != nil {
		return money.Money{}, nil, err
	}
	if toAmount, err = money.FromNullInput(in.ToAmount, toUnit); err != nil {
		return money.Money{}, nil, err
	}
	if toAmount != nil || in.Type != "transfer" || in.ToAccountID == nil {
		return amount, toAmount, nil
	}
	if unit == toUnit {
		return amount, &amount, nil
	}
	accts := Accounts{DB: s.DB}
	from, err := accts.ByID(ctx, in.AccountID)
	if err != nil || from == nil {
		return money.Money{}, nil, ErrUnknownAccount
	}
	to, err := accts.ByID(ctx, *in.ToAccountID)
	if err != nil || to == nil {
		return money.Money{}, nil, ErrUnknownAccount
	}
	converted, err := TryConvert(amount, *from.Currency, *to.Currency)
	if err != nil {
		return money.Money{}, nil, err
	}
	return amount, &converted, nil
}

func txFrom(r repo.Tx) Transaction {
	return Transaction{
		ID: r.ID, Type: r.Type, AccountID: r.AccountID, ToAccountID: r.ToAccountID, CategoryID: r.CategoryID,
		Amount: r.Amount, ToAmount: r.ToAmount, IsEstimated: r.IsEstimated, Description: r.Description, Date: r.Date,
		Status: r.Status, RecurringID: r.RecurringID, CreatedAt: r.CreatedAt,
		GroupUUID: r.GroupUUID,
	}
}

// keepsEstimate tells whether a transaction may carry an approximate amount:
// only a pending income, expense or transfer without items, since items fix
// the amount and the other types settle debts whose amounts are known.
func keepsEstimate(txType, status string, items int) bool {
	if status != "pending" || items > 0 {
		return false
	}
	return txType == "income" || txType == "expense" || txType == "transfer"
}

// IsFuture compares plain "YYYY-MM-DD" calendar dates as strings, using
// appLocation's day as "today". A time.Time-based comparison would parse the
// date at UTC midnight while "now" is truncated separately, so a client whose
// local day has already rolled over (e.g. UTC+3 just after midnight) could
// have a same-day date wrongly rejected as future. Only the first 10 chars
// are used so legacy rows stored as "YYYY-MM-DD HH:MM:SS" still compare correctly.
func IsFuture(date string) bool {
	if len(date) < 10 {
		return false
	}
	today := time.Now().In(appLocation).Format("2006-01-02")
	return date[:10] > today
}
