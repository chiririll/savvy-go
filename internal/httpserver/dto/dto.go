// Package dto converts domain types to HTTP response bodies.
// Keeping serialization here lets domain types stay free of JSON/HTTP concerns.
//
// Conventions: keys are camelCase; optional values are always present and null
// when unset; money is a JSON number at the scale the domain stored it with.
package dto

import (
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/domain"
	"savvy-go/internal/money"
)

func init() {
	// Money goes over the wire as a JSON number, not a string. Values read from
	// storage already carry their currency's scale; computed totals are rounded
	// to it in the domain.
	decimal.MarshalJSONWithoutQuotes = true
}

// Map converts every element of in with fn. It never returns nil, so empty
// lists serialize as [].
func Map[T, R any](in []T, fn func(T) R) []R {
	out := make([]R, 0, len(in))
	for _, v := range in {
		out = append(out, fn(v))
	}
	return out
}

// ptr converts an optional domain value; nil stays nil.
func ptr[T, R any](v *T, fn func(T) R) *R {
	if v == nil {
		return nil
	}
	r := fn(*v)
	return &r
}

func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

type Currency struct {
	ID       int64           `json:"id"`
	Code     string          `json:"code"`
	Name     string          `json:"name"`
	Symbol   string          `json:"symbol"`
	Decimals int             `json:"decimals"`
	IsBase   bool            `json:"isBase"`
	Rate     decimal.Decimal `json:"rate"`
}

func NewCurrency(c domain.Currency) Currency {
	return Currency{ID: c.ID, Code: c.Code, Name: c.Name, Symbol: c.Symbol, Decimals: c.Decimals, IsBase: c.IsBase, Rate: c.Rate}
}

type Tag struct {
	ID                int64      `json:"id"`
	Name              string     `json:"name"`
	TransactionsCount int        `json:"transactionsCount"`
	CreatedAt         *time.Time `json:"createdAt"`
}

func NewTag(t domain.Tag) Tag {
	return Tag{ID: t.ID, Name: t.Name, TransactionsCount: t.TransactionsCount, CreatedAt: utc(t.CreatedAt)}
}

type Category struct {
	ID                int64        `json:"id"`
	Name              string       `json:"name"`
	Type              string       `json:"type"`
	Icon              *string      `json:"icon"`
	Color             *string      `json:"color"`
	IsDefault         bool         `json:"isDefault"`
	TransactionsCount int          `json:"transactionsCount"`
	TotalAmount       *money.Money `json:"totalAmount"`
}

func NewCategory(c domain.Category) Category {
	return Category{
		ID: c.ID, Name: c.Name, Type: c.Type, Icon: c.Icon, Color: c.Color, IsDefault: c.IsDefault,
		TransactionsCount: c.TransactionsCount, TotalAmount: c.TotalAmount,
	}
}

type Account struct {
	ID             int64       `json:"id"`
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	InitialBalance money.Money `json:"initialBalance"`
	CurrentBalance money.Money `json:"currentBalance"`
	IsActive       bool        `json:"isActive"`
	SortOrder      int         `json:"sortOrder"`
	Currency       *Currency   `json:"currency"`
	CreatedAt      *time.Time  `json:"createdAt"`
}

func NewAccount(a domain.Account) Account {
	return Account{
		ID: a.ID, Name: a.Name, Type: a.Type,
		InitialBalance: a.InitialBalance, CurrentBalance: a.Balance,
		IsActive: a.IsActive, SortOrder: a.SortOrder,
		Currency: ptr(a.Currency, NewCurrency), CreatedAt: utc(a.CreatedAt),
	}
}

// Debt is a debt account. CurrentBalance is the amount still owed.
type Debt struct {
	Account
	DebtType        *string     `json:"debtType"`
	TargetAmount    money.Money `json:"targetAmount"`
	PaymentProgress float64     `json:"paymentProgress"`
	DueDate         *string     `json:"dueDate"`
	Counterparty    *string     `json:"counterparty"`
	Description     *string     `json:"description"`
	IsPaidOff       bool        `json:"isPaidOff"`
}

func NewDebt(a domain.Account) Debt {
	target := money.Zero(a.Currency.Unit())
	if a.TargetAmount != nil {
		target = *a.TargetAmount
	}
	progress := 0.0
	if target.IsPositive() {
		paid := target.Sub(a.Balance)
		progress = max(0, 100*float64(paid.Minor())/float64(target.Minor()))
	}
	return Debt{
		Account: NewAccount(a), DebtType: a.DebtType, TargetAmount: target, PaymentProgress: progress,
		DueDate: a.DueDate, Counterparty: a.Counterparty, Description: a.DebtDesc, IsPaidOff: a.IsPaidOff,
	}
}

type TxItem struct {
	ID           int64           `json:"id"`
	Name         string          `json:"name"`
	Quantity     decimal.Decimal `json:"quantity"`
	PricePerUnit money.Money     `json:"pricePerUnit"`
	TotalPrice   money.Money     `json:"totalPrice"`
}

func NewTxItem(i domain.TxItem) TxItem {
	return TxItem{ID: i.ID, Name: i.Name, Quantity: i.Quantity, PricePerUnit: i.PricePerUnit, TotalPrice: i.TotalPrice}
}

type TxActions struct {
	Edit      bool `json:"edit"`
	Delete    bool `json:"delete"`
	Duplicate bool `json:"duplicate"`
	Confirm   bool `json:"confirm"`
	Skip      bool `json:"skip"`
}

type Transaction struct {
	ID                     int64        `json:"id"`
	Type                   string       `json:"type"`
	Amount                 money.Money  `json:"amount"`
	ToAmount               *money.Money `json:"toAmount"`
	IsEstimated            bool         `json:"isEstimated"`
	Description            *string      `json:"description"`
	Date                   *string      `json:"date"`
	Status                 string       `json:"status"`
	RecurringTransactionID *int64       `json:"recurringTransactionId"`
	Actions                TxActions    `json:"actions"`
	Account                *Account     `json:"account"`
	ToAccount              *Account     `json:"toAccount"`
	Category               *Category    `json:"category"`
	Items                  []TxItem     `json:"items"`
	Tags                   []Tag        `json:"tags"`
	CreatedAt              *time.Time   `json:"createdAt"`
}

func NewTransaction(t domain.Transaction) Transaction {
	return Transaction{
		ID: t.ID, Type: t.Type, Amount: t.Amount, ToAmount: t.ToAmount, IsEstimated: t.IsEstimated,
		Description: t.Description, Date: t.Date, Status: t.Status, RecurringTransactionID: t.RecurringID,
		Actions:   transactionActions(t),
		Account:   ptr(t.Account, NewAccount),
		ToAccount: ptr(t.ToAccount, NewAccount),
		Category:  ptr(t.Category, NewCategory),
		Items:     Map(t.Items, NewTxItem),
		Tags:      Map(t.Tags, NewTag),
		CreatedAt: utc(t.CreatedAt),
	}
}

func transactionActions(t domain.Transaction) TxActions {
	// One side of a transfer between spaces changes only through the
	// transfer (in the space settings), never as a transaction.
	if t.Type == "transfer_out" || t.Type == "transfer_in" {
		return TxActions{}
	}
	pending := t.Status == "pending"
	skipped := t.Status == "skipped"
	recurring := t.RecurringID != nil
	return TxActions{
		Edit:      !skipped && !recurring,
		Delete:    !recurring,
		Duplicate: !recurring && !skipped,
		Confirm:   pending,
		Skip:      pending && recurring,
	}
}

type TransactionSummary struct {
	Income            money.Money `json:"income"`
	Expense           money.Money `json:"expense"`
	Balance           money.Money `json:"balance"`
	TransactionsCount int         `json:"transactionsCount"`
	Currency          *string     `json:"currency"`
}

func NewTransactionSummary(s domain.TransactionSummary) TransactionSummary {
	return TransactionSummary{
		Income: s.Income, Expense: s.Expense, Balance: s.Income.Sub(s.Expense),
		TransactionsCount: s.Count, Currency: currencyCode(s.Currency),
	}
}

type BudgetProgress struct {
	Spent       money.Money `json:"spent"`
	Remaining   money.Money `json:"remaining"`
	Percent     float64     `json:"percent"`
	PeriodStart string      `json:"periodStart"`
	PeriodEnd   string      `json:"periodEnd"`
	IsExceeded  bool        `json:"isExceeded"`
}

func NewBudgetProgress(p domain.BudgetProgress) BudgetProgress {
	return BudgetProgress{
		Spent: p.Spent, Remaining: p.Remaining, Percent: p.Percent,
		PeriodStart: p.PeriodStart, PeriodEnd: p.PeriodEnd, IsExceeded: p.IsExceeded,
	}
}

type Budget struct {
	ID              int64           `json:"id"`
	Name            string          `json:"name"`
	Amount          money.Money     `json:"amount"`
	Currency        *Currency       `json:"currency"`
	Period          string          `json:"period"`
	StartDate       *string         `json:"startDate"`
	EndDate         *string         `json:"endDate"`
	IsGlobal        bool            `json:"isGlobal"`
	NotifyAtPercent *int            `json:"notifyAtPercent"`
	IsActive        bool            `json:"isActive"`
	Categories      []Category      `json:"categories"`
	Tags            []Tag           `json:"tags"`
	Progress        *BudgetProgress `json:"progress"`
}

func NewBudget(b domain.Budget) Budget {
	return Budget{
		ID: b.ID, Name: b.Name, Amount: b.Amount, Currency: ptr(b.Currency, NewCurrency),
		Period: b.Period, StartDate: b.StartDate, EndDate: b.EndDate,
		IsGlobal: b.IsGlobal, NotifyAtPercent: b.NotifyAtPercent, IsActive: b.IsActive,
		Categories: Map(b.Categories, NewCategory),
		Tags:       Map(b.Tags, NewTag),
		Progress:   ptr(b.Progress, NewBudgetProgress),
	}
}

type Recurring struct {
	ID          int64        `json:"id"`
	Type        string       `json:"type"`
	Amount      money.Money  `json:"amount"`
	ToAmount    *money.Money `json:"toAmount"`
	IsEstimated bool         `json:"isEstimated"`
	Description *string      `json:"description"`
	Frequency   string       `json:"frequency"`
	Interval    int          `json:"interval"`
	DayOfWeek   *int         `json:"dayOfWeek"`
	DayOfMonth  *int         `json:"dayOfMonth"`
	StartDate   string       `json:"startDate"`
	EndDate     *string      `json:"endDate"`
	NextRunDate string       `json:"nextRunDate"`
	LastRunDate *string      `json:"lastRunDate"`
	IsActive    bool         `json:"isActive"`
	Account     *Account     `json:"account"`
	ToAccount   *Account     `json:"toAccount"`
	Category    *Category    `json:"category"`
	Tags        []Tag        `json:"tags"`
}

func NewRecurring(r domain.Recurring) Recurring {
	return Recurring{
		ID: r.ID, Type: r.Type, Amount: r.Amount, ToAmount: r.ToAmount, IsEstimated: r.IsEstimated, Description: r.Description,
		Frequency: r.Frequency, Interval: r.Interval, DayOfWeek: r.DayOfWeek, DayOfMonth: r.DayOfMonth,
		StartDate: r.StartDate, EndDate: r.EndDate, NextRunDate: r.NextRunDate, LastRunDate: r.LastRunDate,
		IsActive:  r.IsActive,
		Account:   ptr(r.Account, NewAccount),
		ToAccount: ptr(r.ToAccount, NewAccount),
		Category:  ptr(r.Category, NewCategory),
		Tags:      Map(r.Tags, NewTag),
	}
}

type AutomationRule struct {
	ID             int64            `json:"id"`
	Name           string           `json:"name"`
	Description    *string          `json:"description"`
	TriggerType    string           `json:"triggerType"`
	Priority       int              `json:"priority"`
	Conditions     map[string]any   `json:"conditions"`
	Actions        []map[string]any `json:"actions"`
	IsActive       bool             `json:"isActive"`
	StopProcessing bool             `json:"stopProcessing"`
	RunsCount      int              `json:"runsCount"`
	LastRunAt      *time.Time       `json:"lastRunAt"`
	CreatedAt      *time.Time       `json:"createdAt"`
	UpdatedAt      *time.Time       `json:"updatedAt"`
}

func NewAutomationRule(r domain.AutomationRule) AutomationRule {
	return AutomationRule{
		ID: r.ID, Name: r.Name, Description: r.Description, TriggerType: r.TriggerType, Priority: r.Priority,
		Conditions: r.Conditions, Actions: r.Actions, IsActive: r.IsActive, StopProcessing: r.StopProcessing,
		RunsCount: r.RunsCount, LastRunAt: utc(r.LastRunAt), CreatedAt: utc(r.CreatedAt), UpdatedAt: utc(r.UpdatedAt),
	}
}

type AutomationLog struct {
	ID                int64     `json:"id"`
	RuleID            int64     `json:"ruleId"`
	TriggerEntityType *string   `json:"triggerEntityType"`
	TriggerEntityID   *int64    `json:"triggerEntityId"`
	ActionsExecuted   any       `json:"actionsExecuted"`
	Status            string    `json:"status"`
	ErrorMessage      *string   `json:"errorMessage"`
	CreatedAt         time.Time `json:"createdAt"`
}

func NewAutomationLog(l domain.AutomationLog) AutomationLog {
	return AutomationLog{
		ID: l.ID, RuleID: l.RuleID, TriggerEntityType: l.TriggerEntityType, TriggerEntityID: l.TriggerEntityID,
		ActionsExecuted: l.ActionsExecuted, Status: l.Status, ErrorMessage: l.ErrorMessage, CreatedAt: l.CreatedAt.UTC(),
	}
}

type AccountsSummary struct {
	TotalBalance  money.Money `json:"totalBalance"`
	Currency      *string     `json:"currency"`
	Decimals      int         `json:"decimals"`
	AccountsCount int         `json:"accountsCount"`
}

func NewAccountsSummary(s domain.AccountsSummary) AccountsSummary {
	return AccountsSummary{
		TotalBalance: s.Total, Currency: currencyCode(s.Currency), Decimals: currencyDecimals(s.Currency), AccountsCount: s.Count,
	}
}

type DebtSummary struct {
	TotalIOwe     money.Money `json:"totalIOwe"`
	TotalOwedToMe money.Money `json:"totalOwedToMe"`
	NetDebt       money.Money `json:"netDebt"`
	DebtsCount    int         `json:"debtsCount"`
	Currency      *string     `json:"currency"`
	Decimals      int         `json:"decimals"`
}

func NewDebtSummary(s domain.DebtSummary) DebtSummary {
	return DebtSummary{
		TotalIOwe: s.IOwe, TotalOwedToMe: s.OwedToMe, NetDebt: s.OwedToMe.Sub(s.IOwe), DebtsCount: s.Count,
		Currency: currencyCode(s.Currency), Decimals: currencyDecimals(s.Currency),
	}
}

type CategoryStatistics struct {
	CategoryID        int64       `json:"categoryId"`
	CategoryName      string      `json:"categoryName"`
	Type              string      `json:"type"`
	TransactionsCount int         `json:"transactionsCount"`
	TotalAmount       money.Money `json:"totalAmount"`
}

func NewCategoryStatistics(s domain.CategoryStatistics) CategoryStatistics {
	return CategoryStatistics{
		CategoryID: s.Category.ID, CategoryName: s.Category.Name, Type: s.Category.Type,
		TransactionsCount: s.Count, TotalAmount: s.Total,
	}
}

func currencyCode(c *domain.Currency) *string {
	if c == nil {
		return nil
	}
	return &c.Code
}

// currencyDecimals is the scale totals in c are rounded to (2 without one).
func currencyDecimals(c *domain.Currency) int {
	if c == nil {
		return 2
	}
	return c.Decimals
}

type ImportFormats struct {
	DateFormat   any  `json:"dateFormat"`
	AmountFormat any  `json:"amountFormat"`
	HasHeader    bool `json:"hasHeader"`
	Delimiter    any  `json:"delimiter"`
}

type ImportParse struct {
	Headers          any           `json:"headers"`
	PreviewRows      any           `json:"previewRows"`
	TotalRows        *int          `json:"totalRows"`
	DetectedFormats  ImportFormats `json:"detectedFormats"`
	SuggestedMapping any           `json:"suggestedMapping"`
}

type ImportResult struct {
	Created           int `json:"created"`
	SkippedDuplicates int `json:"skippedDuplicates"`
	Errors            any `json:"errors"`
	CreatedCurrencies any `json:"createdCurrencies"`
	CreatedTags       any `json:"createdTags"`
	CreatedCategories any `json:"createdCategories"`
}

type Import struct {
	ImportID      string        `json:"importId"`
	Status        string        `json:"status"`
	TotalRows     *int          `json:"totalRows"`
	ProcessedRows int           `json:"processedRows"`
	Created       int           `json:"created"`
	Skipped       int           `json:"skipped"`
	Errors        int           `json:"errors"`
	Message       *string       `json:"message"`
	Parse         *ImportParse  `json:"parse"`
	Result        *ImportResult `json:"result"`
}

// NewImport reads the parse/result details from im.Meta, which the domain
// stores as JSON with snake_case keys.
func NewImport(im domain.Import) Import {
	out := Import{
		ImportID: im.ID, Status: im.Status, TotalRows: im.TotalRows, ProcessedRows: im.ProcessedRows,
		Created: im.CreatedCount, Skipped: im.SkippedCount, Errors: im.ErrorCount, Message: im.Message,
	}
	parsed := im.Status == "parsed" || im.Status == "importing" || im.Status == "completed"
	if parsed && im.Meta != nil {
		formats, _ := im.Meta["detected_formats"].(map[string]any)
		hasHeader, _ := formats["has_header"].(bool)
		out.Parse = &ImportParse{
			Headers:     im.Meta["headers"],
			PreviewRows: im.Meta["preview_rows"],
			TotalRows:   im.TotalRows,
			DetectedFormats: ImportFormats{
				DateFormat: formats["date_format"], AmountFormat: formats["amount_format"],
				HasHeader: hasHeader, Delimiter: formats["delimiter"],
			},
			SuggestedMapping: im.Meta["suggested_mapping"],
		}
	}
	if im.Status == "completed" {
		out.Result = &ImportResult{
			Created:           im.CreatedCount,
			SkippedDuplicates: im.SkippedCount,
			Errors:            im.Errors,
			CreatedCurrencies: im.Meta["created_currencies"],
			CreatedTags:       im.Meta["created_tags"],
			CreatedCategories: im.Meta["created_categories"],
		}
	}
	return out
}

type WebAuthnCred struct {
	ID         int64   `json:"id"`
	Name       *string `json:"name"`
	AAGUID     *string `json:"aaguid"`
	LastUsedAt *string `json:"lastUsedAt"`
	CreatedAt  *string `json:"createdAt"`
}

func NewWebAuthnCred(c auth.WebAuthnCred) WebAuthnCred {
	return WebAuthnCred{ID: c.ID, Name: c.Name, AAGUID: c.AAGUID, LastUsedAt: c.LastUsedAt, CreatedAt: c.CreatedAt}
}

// Backup status: "current" is a backup signed by this server (or a key it
// trusts), "unsigned" one made elsewhere or edited, and "invalid" one that cannot
// be read.
type Backup struct {
	Filename   string     `json:"filename"`
	Size       int64      `json:"size"`
	Kind       string     `json:"kind"`
	Note       *string    `json:"note"`
	AppVersion *string    `json:"appVersion"`
	SpaceName  string     `json:"spaceName,omitempty"`
	Status     string     `json:"status"`
	Signature  string     `json:"signature"`
	Restorable bool       `json:"restorable"`
	CreatedAt  *time.Time `json:"createdAt"`
}

func NewBackup(b domain.Backup) Backup {
	status := "invalid"
	switch {
	case !b.Valid:
	case b.Signature == domain.SignedHere || b.Signature == domain.SignedTrusted:
		status = "current"
	default:
		status = "unsigned"
	}
	return Backup{
		Filename: b.Filename, Size: b.Size, Kind: b.Kind, Note: b.Note, AppVersion: b.AppVersion,
		SpaceName: b.SpaceName, Status: status, Signature: string(b.Signature), Restorable: b.Valid,
		CreatedAt: utc(b.CreatedAt),
	}
}

func APIToken(t auth.APIToken) map[string]any {
	return map[string]any{
		"id": t.ID, "name": t.Name, "prefix": t.Prefix, "scope": t.Scope,
		"expires_at": timePtr(t.ExpiresAt), "last_used_at": timePtr(t.LastUsedAt), "created_at": timePtr(t.CreatedAt),
	}
}

func timePtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}
