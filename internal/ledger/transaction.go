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

// PaymentInput is money that moves on one account, signed, in the currency of
// the account. Amount is nil only in a recurring template, where the payment
// takes whatever balances the others at each occurrence; Estimated marks such
// an amount as a guess until the occurrence is confirmed.
type PaymentInput struct {
	AccountID int64            `json:"account_id"`
	Amount    *decimal.Decimal `json:"amount"`
	Estimated bool             `json:"estimated,omitempty"`

	// SpaceTransferUUID is set only by internal/transfers, on the payment on a
	// space account that carries a transfer between spaces.
	SpaceTransferUUID *string `json:"-"`
}

// LineInput is an amount for a category, positive, in one of the currencies of
// the payment accounts. A zero CurrencyID means the currency all the payment
// accounts share. A nil CategoryID is uncategorised.
type LineInput struct {
	CategoryID *int64          `json:"category_id"`
	CurrencyID int64           `json:"currency_id"`
	Amount     decimal.Decimal `json:"amount"`
	Note       *string         `json:"note"`
}

// TxInput is a transaction as entered, and the template of a recurring rule:
// the same JSON, the same type. Payments and Lines left nil mean "as they are"
// on update; TagIDs likewise.
type TxInput struct {
	Type        string         `json:"type"`
	Description *string        `json:"description"`
	Date        *string        `json:"date"`
	Status      *string        `json:"status,omitempty"`
	Payments    []PaymentInput `json:"payments"`
	Lines       []LineInput    `json:"lines"`
	TagIDs      []int64        `json:"tag_ids"`

	// ExternalID and CreatedBy are written when the transaction is created.
	ExternalID *string `json:"-"`
	CreatedBy  *int64  `json:"-"`
}

// Payment is money that moved on Account; negative when it left.
type Payment struct {
	AccountID         int64
	Account           *Account
	Amount            money.Money // in the currency of Account
	SpaceTransferUUID *string
}

// Line is an amount for a category, in the currency of Amount.
type Line struct {
	Seq        int
	CategoryID *int64
	Category   *Category
	Amount     money.Money
	Note       *string
}

// Transaction is one event with the payments that moved the money and the
// lines that say what it was for. Nothing ties the two by constraint; Difference
// reports where they disagree.
type Transaction struct {
	ID          int64
	Type        string
	Description *string
	Date        *string
	Status      string
	ExternalID  *string
	CreatedAt   *time.Time
	RecurringID *int64 // the rule whose occurrence this is
	Payments    []Payment
	Lines       []Line
	Tags        []Tag
	Reasons     []Reason
	// Difference is what the payments moved less what the lines account for, in
	// the base currency at the current rates. It is set only on the result of
	// Create and Update when payments or lines changed, and nil when they agree.
	Difference *money.Money
}

// Amount is the one figure a rule or a list can compare: for an income or
// expense what the payments net to, for any other type what left the accounts,
// always positive. It is false when the payments are in several currencies,
// which cannot be added without a rate.
func (t Transaction) Amount() (money.Money, bool) {
	var total money.Money
	found := false
	for _, p := range t.Payments {
		if t.Type != "income" && t.Type != "expense" && !p.Amount.IsNegative() {
			continue
		}
		if !found {
			total, found = p.Amount, true
			continue
		}
		if p.Amount.Unit() != total.Unit() {
			return money.Money{}, false
		}
		total = total.Add(p.Amount)
	}
	return total.Abs(), found
}

// Input is the transaction as an update input, for read-modify-write.
func (t Transaction) Input() TxInput {
	in := TxInput{Type: t.Type, Description: t.Description, Date: t.Date}
	for _, p := range t.Payments {
		amount := p.Amount.Decimal()
		in.Payments = append(in.Payments, PaymentInput{AccountID: p.AccountID, Amount: &amount})
	}
	for _, l := range t.Lines {
		in.Lines = append(in.Lines, LineInput{
			CategoryID: l.CategoryID, CurrencyID: l.Amount.Unit().ID, Amount: l.Amount.Decimal(), Note: l.Note,
		})
	}
	for _, g := range t.Tags {
		in.TagIDs = append(in.TagIDs, g.ID)
	}
	return in
}

// without is the transaction as its owner sees it: the owner's own reason does
// not restrict it.
func (t Transaction) without(owner Reason) Transaction {
	if owner == "" {
		return t
	}
	t.Reasons = slices.DeleteFunc(slices.Clone(t.Reasons), func(r Reason) bool { return r == owner })
	return t
}

// Transactions is the service for transactions. As narrows it to the module
// that owns a kind of transaction: that module alone may write the service
// accounts its reason covers, and its own reason does not restrict it.
type Transactions struct {
	DB store.DB
	as Reason
}

func (s Transactions) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// As is the service as used by the module that owns reason: debts for
// ReasonDebt, transfers for ReasonSpaceTransfer, recurring for
// ReasonRecurringPending.
func (s Transactions) As(reason Reason) Transactions { return Transactions{DB: s.DB, as: reason} }

// On is the service on another handle, for code inside a database transaction.
func (s Transactions) On(d store.DB) Transactions { return Transactions{DB: d, as: s.as} }

// ownerOf is the module whose service accounts a change to cur may keep: the
// caller's own, or the one that owns the transaction. Keeping is not changing:
// Update checks separately that only the details, or what the owner left
// open, differ from the stored transaction.
func (s Transactions) ownerOf(cur Transaction) Reason {
	if s.as != "" {
		return s.as
	}
	for _, r := range cur.Reasons {
		if r == ReasonDebt || r == ReasonSpaceTransfer {
			return r
		}
	}
	return ""
}

// built is a transaction input resolved against the database: the rows to
// store and the shape the rules check.
type built struct {
	shape    shape
	payments []repo.PaymentFields
	lines    []repo.LineFields
}

// build resolves an input: loads its accounts, categories and currencies,
// rounds amounts into their currencies, and checks the shape.
func (s Transactions) build(ctx context.Context, in TxInput, template bool) (built, error) {
	b := built{shape: shape{Type: in.Type, Owner: s.as, Template: template}}
	accts := Accounts{DB: s.DB}
	currencies := map[int64]Currency{}
	for _, p := range in.Payments {
		a, err := accts.ByID(ctx, p.AccountID)
		if err != nil {
			return built{}, err
		}
		if a == nil || a.Currency == nil {
			return built{}, ErrUnknownAccount
		}
		currencies[a.Currency.ID] = *a.Currency
		sp := shapePayment{Account: *a}
		if p.SpaceTransferUUID != nil && (s.as != ReasonSpaceTransfer || a.Type != "space") {
			return built{}, fmt.Errorf("%w: only a transfer between spaces sets this", ErrSpaceShape)
		}
		if p.Amount == nil {
			sp.Balancing = true
		} else {
			if sp.Amount, err = money.FromInput(*p.Amount, a.Currency.Unit()); err != nil {
				return built{}, err
			}
			b.payments = append(b.payments, repo.PaymentFields{
				AccountID: a.ID, Amount: sp.Amount, SpaceTransferUUID: p.SpaceTransferUUID,
			})
		}
		b.shape.Payments = append(b.shape.Payments, sp)
	}
	curs, cats := Currencies{DB: s.DB}, Categories{DB: s.DB}
	for _, l := range in.Lines {
		cur, ok := currencies[l.CurrencyID]
		if l.CurrencyID == 0 && len(currencies) == 1 {
			for _, only := range currencies {
				cur, ok = only, true
			}
		}
		if !ok { // a currency no payment account has: let the rules say so
			c, err := curs.ByID(ctx, l.CurrencyID)
			if err != nil {
				return built{}, err
			}
			if c == nil {
				return built{}, ErrUnknownCurrency
			}
			cur = *c
		}
		amount, err := money.FromInput(l.Amount, cur.Unit())
		if err != nil {
			return built{}, err
		}
		sl := shapeLine{Currency: cur, Amount: amount}
		if l.CategoryID != nil {
			if sl.Category, err = cats.ByID(ctx, *l.CategoryID); err != nil {
				return built{}, err
			}
			if sl.Category == nil {
				return built{}, ErrUnknownCategory
			}
		}
		b.shape.Lines = append(b.shape.Lines, sl)
		b.lines = append(b.lines, repo.LineFields{CategoryID: l.CategoryID, Amount: amount, Note: l.Note})
	}
	return b, b.shape.check()
}

// ConvertedTransfer is the input of a transfer of amount, in the currency of
// from, that delivers the same value on to at the current rates.
func (s Transactions) ConvertedTransfer(ctx context.Context, from, to int64, amount decimal.Decimal) (TxInput, error) {
	accts := Accounts{DB: s.DB}
	src, err := accts.ByID(ctx, from)
	if err != nil || src == nil || src.Currency == nil {
		return TxInput{}, ErrUnknownAccount
	}
	dst, err := accts.ByID(ctx, to)
	if err != nil || dst == nil || dst.Currency == nil {
		return TxInput{}, ErrUnknownAccount
	}
	out, err := money.FromInput(amount, src.Currency.Unit())
	if err != nil {
		return TxInput{}, err
	}
	in, err := TryConvert(out, *src.Currency, *dst.Currency)
	if err != nil {
		return TxInput{}, err
	}
	taken, delivered := out.Neg().Decimal(), in.Decimal()
	return TxInput{
		Type:     "transfer",
		Payments: []PaymentInput{{AccountID: from, Amount: &taken}, {AccountID: to, Amount: &delivered}},
	}, nil
}

// ErrUnknownCategory is returned when a category referenced by id does not exist.
var ErrUnknownCategory = errors.New("unknown category")

// difference is the soft check on a built transaction: nil when payments and
// lines agree, or when there is no base currency to compare in.
func (s Transactions) difference(ctx context.Context, b built) *money.Money {
	base, err := (Currencies{DB: s.DB}).Base(ctx)
	if err != nil || base == nil || len(b.shape.Payments) == 0 {
		return nil
	}
	d, ok := b.shape.difference(*base)
	if !ok || d.IsZero() {
		return nil
	}
	return &d
}

func (s Transactions) Create(ctx context.Context, in TxInput) (*Transaction, error) {
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
	b, err := s.build(ctx, in, false)
	if err != nil {
		return nil, err
	}
	var id int64
	err = store.Tx(ctx, s.DB, func(d store.DB) error {
		r := repo.Repo{DB: d}
		id, err = r.InsertTransaction(ctx, repo.TxFields{
			Type: in.Type, Status: status, Description: blankToNil(in.Description), Date: in.Date,
			ExternalID: in.ExternalID, CreatedBy: in.CreatedBy,
		}, time.Now().UTC())
		if err != nil {
			return err
		}
		return s.store(ctx, d, id, b, in.TagIDs)
	})
	if err != nil {
		return nil, err
	}
	tx, err := s.ByID(ctx, id)
	if tx != nil {
		tx.Difference = s.difference(ctx, b)
	}
	return tx, err
}

// store writes the payments, lines and tags of a transaction.
func (s Transactions) store(ctx context.Context, d store.DB, id int64, b built, tagIDs []int64) error {
	r := repo.Repo{DB: d}
	if err := r.ReplacePayments(ctx, id, b.payments); err != nil {
		return err
	}
	if err := r.ReplaceLines(ctx, id, b.lines); err != nil {
		return err
	}
	if tagIDs != nil {
		return r.SetTags(ctx, id, tagIDs)
	}
	return nil
}

// blankToNil stores an empty description as none.
func blankToNil(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// ErrDetailsOnly is returned when a change to a transaction that allows only
// its details touches the money, the date or the type.
var ErrDetailsOnly = errors.New("only the description, tags and the categories and notes of lines can change")

func (s Transactions) Update(ctx context.Context, id int64, in TxInput) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	seen := cur.without(s.as)
	// What the caller leaves out is kept; an empty description clears it.
	merged := cur.Input()
	if in.Payments != nil {
		merged.Payments = in.Payments
	}
	if in.Lines != nil {
		merged.Lines = in.Lines
	}
	if in.Type != "" {
		merged.Type = in.Type
	}
	if in.Date != nil {
		merged.Date = in.Date
	}
	if in.Description != nil {
		merged.Description = in.Description
	}
	merged.TagIDs = in.TagIDs
	in = merged
	if seen.Status == "confirmed" && in.Date == nil {
		return nil, fmt.Errorf("date required")
	}
	b, err := s.As(s.ownerOf(*cur)).build(ctx, in, false)
	if err != nil {
		return nil, err
	}
	changed := !samePayments(cur.Payments, b.payments) || !sameLines(cur.Lines, b.lines, true)
	switch {
	case seen.Allowed().Edit:
		if err := frozenParts(seen, in, b); err != nil {
			return nil, err
		}
	case seen.Allowed().EditDetails:
		if in.Type != cur.Type || !sameDate(cur.Date, in.Date) || !samePayments(cur.Payments, b.payments) ||
			!sameLines(cur.Lines, b.lines, true) {
			return nil, ErrDetailsOnly
		}
	default:
		return nil, seen.refuse(ActionEdit)
	}
	err = store.Tx(ctx, s.DB, func(d store.DB) error {
		if err := (repo.Repo{DB: d}).UpdateTransaction(ctx, id, repo.TxFields{
			Type: in.Type, Description: blankToNil(in.Description), Date: in.Date,
		}, time.Now().UTC()); err != nil {
			return err
		}
		return s.store(ctx, d, id, b, in.TagIDs)
	})
	if err != nil {
		return nil, err
	}
	tx, err := s.ByID(ctx, id)
	if tx != nil && changed {
		tx.Difference = s.difference(ctx, b)
	}
	return tx, err
}

// frozenParts keeps what only the owner of a transfer between spaces changes:
// the payment on the space account, the date and the type, and for a transfer
// the real payment it pairs with.
func frozenParts(cur Transaction, in TxInput, b built) error {
	if !slices.Contains(cur.Reasons, ReasonSpaceTransfer) {
		return nil
	}
	if in.Type != cur.Type || !sameDate(cur.Date, in.Date) {
		return cur.refuse(ActionSpacePayment)
	}
	paired := cur.Type == "transfer_out" || cur.Type == "transfer_in"
	for _, p := range cur.Payments {
		if p.SpaceTransferUUID == nil && !paired {
			continue
		}
		if !hasPayment(b.payments, p) {
			return cur.refuse(ActionSpacePayment)
		}
	}
	return nil
}

func hasPayment(have []repo.PaymentFields, p Payment) bool {
	return slices.ContainsFunc(have, func(h repo.PaymentFields) bool {
		return h.AccountID == p.AccountID && h.Amount.Equal(p.Amount)
	})
}

func samePayments(cur []Payment, next []repo.PaymentFields) bool {
	if len(cur) != len(next) {
		return false
	}
	for _, p := range cur {
		if !hasPayment(next, p) {
			return false
		}
	}
	return true
}

// sameLines compares lines in order: by amount and currency, and, unless
// moneyOnly, by category and note too.
func sameLines(cur []Line, next []repo.LineFields, moneyOnly bool) bool {
	if len(cur) != len(next) {
		return false
	}
	for i, l := range cur {
		n := next[i]
		if !l.Amount.Equal(n.Amount) {
			return false
		}
		if !moneyOnly && (!sameInt(l.CategoryID, n.CategoryID) || !sameStr(l.Note, n.Note)) {
			return false
		}
	}
	return true
}

func sameInt(a, b *int64) bool   { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
func sameStr(a, b *string) bool  { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
func sameDate(a, b *string) bool { return sameStr(a, b) }

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

func (s Transactions) Delete(ctx context.Context, id int64) error {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return err
	}
	if err := cur.without(s.as).refuse(ActionDelete); err != nil {
		return err
	}
	return s.repo().DeleteTransaction(ctx, id)
}

// Amendment replaces the payments and lines of a pending occurrence when it
// is confirmed: its amounts were a guess (a rate of the day, a bill not yet
// known). Left nil, they stand.
type Amendment struct {
	Payments []PaymentInput
	Lines    []LineInput
}

// Confirm settles a pending transaction on date. Only the open occurrence of
// a recurring rule can be confirmed with other amounts.
func (s Transactions) Confirm(ctx context.Context, id int64, date string, amend *Amendment) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	seen := cur.without(s.as)
	if err := seen.refuse(ActionConfirm); err != nil {
		return nil, err
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
	var b built
	amended := amend != nil && (amend.Payments != nil || amend.Lines != nil)
	if amended {
		if !slices.Contains(cur.Reasons, ReasonRecurringPending) {
			return nil, fmt.Errorf("only a scheduled occurrence can be confirmed with other amounts")
		}
		in := cur.Input()
		if amend.Payments != nil {
			in.Payments = amend.Payments
		}
		if amend.Lines != nil {
			in.Lines = amend.Lines
		}
		if b, err = s.As(s.ownerOf(*cur)).build(ctx, in, false); err != nil {
			return nil, err
		}
	}
	err = store.Tx(ctx, s.DB, func(d store.DB) error {
		r := repo.Repo{DB: d}
		if amended {
			if err := r.ReplacePayments(ctx, id, b.payments); err != nil {
				return err
			}
			if err := r.ReplaceLines(ctx, id, b.lines); err != nil {
				return err
			}
		}
		return r.ConfirmTransaction(ctx, id, date, time.Now().UTC())
	})
	if err != nil {
		return nil, err
	}
	tx, err := s.ByID(ctx, id)
	if tx != nil && amended {
		tx.Difference = s.difference(ctx, b)
	}
	return tx, err
}

func (s Transactions) Skip(ctx context.Context, id int64) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if err := cur.without(s.as).refuse(ActionSkip); err != nil {
		return nil, err
	}
	if err := s.repo().SkipTransaction(ctx, id, time.Now().UTC()); err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// Duplicate copies a transaction as a confirmed one dated today.
func (s Transactions) Duplicate(ctx context.Context, id int64) (*Transaction, error) {
	cur, err := s.ByID(ctx, id)
	if err != nil || cur == nil {
		return cur, err
	}
	if err := cur.without(s.as).refuse(ActionDuplicate); err != nil {
		return nil, err
	}
	in := cur.Input()
	today := Today()
	st := "confirmed"
	in.Date, in.Status = &today, &st
	return Transactions{DB: s.DB}.Create(ctx, in)
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

// SetCategory gives a category to the uncategorised lines of a transaction (an
// automation action). The category must fit the transaction type, as for a
// line entered by hand.
func (s Transactions) SetCategory(ctx context.Context, id, categoryID int64) error {
	tx, err := s.ByID(ctx, id)
	if err != nil || tx == nil {
		return err
	}
	cat, err := (Categories{DB: s.DB}).ByID(ctx, categoryID)
	if err != nil {
		return err
	}
	if cat == nil {
		return ErrUnknownCategory
	}
	if cat.Type != lineCategoryType(tx.Type) {
		return ErrLineCategory
	}
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
