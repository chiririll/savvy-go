package ledger

import (
	"errors"
	"slices"
	"time"

	"github.com/shopspring/decimal"

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

	// ExternalID is the caller's own key for the transaction, such as a receipt
	// number. ExternalSource is set by the server, never read from JSON, and
	// the pair is stored as "source:id". The key is unique in the space:
	// creating a second transaction with it fails with ErrExternalIDExists, so
	// a retry adds nothing. The same id from another source is stored and
	// reported in Transaction.ExternalMatches. It is written when the
	// transaction is created and not changed by an update.
	ExternalSource *string `json:"-"`
	ExternalID     *string `json:"external_id,omitempty"`
	// CreatedBy is written when the transaction is created.
	CreatedBy *int64 `json:"-"`
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
	ID             int64
	Type           string
	Description    *string
	Date           *string
	Status         string
	ExternalSource *string // ui, api, mcp or csv
	ExternalID     *string
	CreatedAt      *time.Time
	RecurringID    *int64 // the rule whose occurrence this is
	Payments       []Payment
	Lines          []Line
	Tags           []Tag
	Reasons        []Reason
	// ExternalMatches is set only on the result of Create: transactions with
	// the same external id under another source.
	ExternalMatches []ExternalMatch
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

// ErrUnknownTransaction is returned for a transaction id that does not exist.
var ErrUnknownTransaction = errors.New("unknown transaction")

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
