// Package dto converts domain types to HTTP response maps.
// Keeping serialization here lets domain types stay free of JSON/HTTP concerns.
package dto

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"savvy-go/internal/auth"
	"savvy-go/internal/domain"
	"savvy-go/internal/money"
	"savvy-go/internal/version"
)

// decimalsOf is the currency decimals of an account (2 when unknown).
func decimalsOf(a *domain.Account) int {
	if a != nil && a.Currency != nil {
		return a.Currency.Decimals
	}
	return 2
}

func MapSlice[T any](in []T, fn func(T) map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, v := range in {
		out = append(out, fn(v))
	}
	return out
}

func Currency(c domain.Currency) map[string]any {
	return map[string]any{
		"id":       c.ID,
		"code":     c.Code,
		"name":     c.Name,
		"symbol":   c.Symbol,
		"decimals": c.Decimals,
		"isBase":   c.IsBase,
		"rate":     money.Plain(c.Rate),
	}
}

func Tag(t domain.Tag) map[string]any {
	var created any
	if t.CreatedAt != nil {
		created = t.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return map[string]any{
		"id":                t.ID,
		"name":              t.Name,
		"transactionsCount": t.TransactionsCount,
		"createdAt":         created,
	}
}

func Category(c domain.Category) map[string]any {
	m := map[string]any{
		"id":        c.ID,
		"name":      c.Name,
		"type":      c.Type,
		"icon":      c.Icon,
		"color":     c.Color,
		"isDefault": c.IsDefault,
	}
	m["transactionsCount"] = c.TransactionsCount
	if c.TotalAmount != nil {
		m["totalAmount"] = money.Plain(*c.TotalAmount)
	}
	return m
}

func Account(a domain.Account) map[string]any {
	var created any
	if a.CreatedAt != nil {
		created = a.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	var cur any
	if a.Currency != nil {
		cur = Currency(*a.Currency)
	}
	dec := decimalsOf(&a)
	return map[string]any{
		"id":             a.ID,
		"name":           a.Name,
		"type":           a.Type,
		"currencyId":     a.CurrencyID,
		"initialBalance": money.Number(a.InitialBalance, dec),
		"currentBalance": money.Number(a.Balance, dec),
		"isActive":       a.IsActive,
		"sortOrder":      a.SortOrder,
		"currency":       cur,
		"createdAt":      created,
	}
}

func TxItem(i domain.TxItem, dec int) map[string]any {
	return map[string]any{
		"id": i.ID, "name": i.Name, "quantity": money.Plain(i.Quantity),
		"pricePerUnit": money.Number(i.PricePerUnit, dec), "totalPrice": money.Number(i.TotalPrice, dec),
	}
}

func Transaction(t domain.Transaction) map[string]any {
	dec := decimalsOf(t.Account)
	toDec := dec
	if t.ToAccount != nil {
		toDec = decimalsOf(t.ToAccount)
	}
	m := map[string]any{
		"id": t.ID, "type": t.Type, "amount": money.Number(t.Amount, dec),
		"description": t.Description, "date": t.Date, "status": t.Status,
		"recurringTransactionId": t.RecurringID,
		"actions":                transactionActions(t),
		"items":                  MapSlice(t.Items, func(i domain.TxItem) map[string]any { return TxItem(i, dec) }),
		"itemsCount":             len(t.Items),
		"tags":                   MapSlice(t.Tags, Tag),
	}
	if t.ToAmount != nil {
		m["toAmount"] = money.Number(*t.ToAmount, toDec)
	}
	if t.ExchangeRate != nil {
		m["exchangeRate"] = money.Plain(*t.ExchangeRate)
	}
	if t.Account != nil {
		m["account"] = Account(*t.Account)
	}
	if t.ToAccount != nil {
		m["toAccount"] = Account(*t.ToAccount)
	}
	if t.Category != nil {
		m["category"] = Category(*t.Category)
	}
	if t.CreatedAt != nil {
		m["createdAt"] = t.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func transactionActions(t domain.Transaction) map[string]bool {
	pending := t.Status == "pending"
	skipped := t.Status == "skipped"
	recurring := t.RecurringID != nil
	return map[string]bool{
		"edit":      !skipped && !recurring,
		"delete":    !recurring,
		"duplicate": !recurring && !skipped,
		"confirm":   pending,
		"skip":      pending && recurring,
	}
}

func BudgetProgress(p domain.BudgetProgress, dec int) map[string]any {
	return map[string]any{
		"spent": money.Number(p.Spent, dec), "remaining": money.Number(p.Remaining, dec), "percent": p.Percent,
		"period_start": p.PeriodStart, "period_end": p.PeriodEnd,
		"is_exceeded": p.IsExceeded,
	}
}

func Budget(b domain.Budget) map[string]any {
	dec := 2
	if b.Currency != nil {
		dec = b.Currency.Decimals
	}
	m := map[string]any{
		"id": b.ID, "name": b.Name, "amount": money.Number(b.Amount, dec),
		"currencyId": b.CurrencyID, "period": b.Period, "periodLabel": budgetPeriodLabel(b.Period),
		"startDate": b.StartDate, "endDate": b.EndDate,
		"isGlobal": b.IsGlobal, "notifyAtPercent": b.NotifyAtPercent,
		"isActive":   b.IsActive,
		"categories": MapSlice(b.Categories, Category),
		"tags":       MapSlice(b.Tags, Tag),
	}
	if b.Currency != nil {
		m["currency"] = Currency(*b.Currency)
	}
	if b.Progress != nil {
		p := BudgetProgress(*b.Progress, dec)
		m["progress"] = p
	}
	return m
}

func budgetPeriodLabel(p string) string {
	switch p {
	case "weekly":
		return "Weekly"
	case "monthly":
		return "Monthly"
	case "quarterly":
		return "Quarterly"
	case "yearly":
		return "Yearly"
	case "one_time":
		return "One-time"
	default:
		return p
	}
}

func Recurring(r domain.Recurring) map[string]any {
	dec := decimalsOf(r.Account)
	toDec := dec
	if r.ToAccount != nil {
		toDec = decimalsOf(r.ToAccount)
	}
	m := map[string]any{
		"id": r.ID, "type": r.Type, "accountId": r.AccountID,
		"amount": money.Number(r.Amount, dec), "description": r.Description,
		"frequency": r.Frequency, "frequencyLabel": recurringFrequencyLabel(r.Frequency),
		"interval": r.Interval, "startDate": r.StartDate,
		"nextRunDate": r.NextRunDate, "isActive": r.IsActive,
		"tags": MapSlice(r.Tags, Tag),
	}
	if r.ToAccountID != nil {
		m["toAccountId"] = *r.ToAccountID
	}
	if r.CategoryID != nil {
		m["categoryId"] = *r.CategoryID
	}
	if r.ToAmount != nil {
		m["toAmount"] = money.Number(*r.ToAmount, toDec)
	}
	if r.DayOfWeek != nil {
		m["dayOfWeek"] = *r.DayOfWeek
	}
	if r.DayOfMonth != nil {
		m["dayOfMonth"] = *r.DayOfMonth
	}
	if r.EndDate != nil {
		m["endDate"] = *r.EndDate
	}
	if r.LastRunDate != nil {
		m["lastRunDate"] = *r.LastRunDate
	}
	if r.Account != nil {
		m["account"] = Account(*r.Account)
	}
	if r.ToAccount != nil {
		m["toAccount"] = Account(*r.ToAccount)
	}
	if r.Category != nil {
		m["category"] = Category(*r.Category)
	}
	return m
}

func recurringFrequencyLabel(f string) string {
	switch f {
	case "daily":
		return "Daily"
	case "weekly":
		return "Weekly"
	case "biweekly":
		return "Biweekly"
	case "monthly":
		return "Monthly"
	case "quarterly":
		return "Quarterly"
	case "yearly":
		return "Yearly"
	default:
		return f
	}
}

func AutomationRule(r domain.AutomationRule) map[string]any {
	label, _ := automationTriggerLabel(r.TriggerType)
	m := map[string]any{
		"id": r.ID, "name": r.Name, "description": r.Description,
		"trigger_type": r.TriggerType, "trigger_label": label,
		"priority": r.Priority, "conditions": r.Conditions, "actions": r.Actions,
		"is_active": r.IsActive, "stop_processing": r.StopProcessing,
		"runs_count": r.RunsCount,
	}
	if r.LastRunAt != nil {
		m["last_run_at"] = r.LastRunAt.UTC().Format(time.RFC3339Nano)
	} else {
		m["last_run_at"] = nil
	}
	if r.CreatedAt != nil {
		m["created_at"] = r.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	if r.UpdatedAt != nil {
		m["updated_at"] = r.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	return m
}

func automationTriggerLabel(t string) (string, string) {
	switch t {
	case "on_transaction_create":
		return "On Transaction Create", "Triggers when a new transaction is created"
	case "on_transaction_update":
		return "On Transaction Update", "Triggers when a transaction is updated"
	default:
		return t, ""
	}
}

func AutomationLog(l domain.AutomationLog) map[string]any {
	return map[string]any{
		"id": l.ID, "rule_id": l.RuleID,
		"trigger_entity_type": l.TriggerEntityType, "trigger_entity_id": l.TriggerEntityID,
		"actions_executed": l.ActionsExecuted, "status": l.Status,
		"error_message": l.ErrorMessage,
		"created_at":    l.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func Import(im domain.Import) map[string]any {
	parsed := im.Status == "parsed" || im.Status == "importing" || im.Status == "completed"
	m := map[string]any{
		"import_id": im.ID, "status": im.Status, "total_rows": im.TotalRows,
		"processed_rows": im.ProcessedRows, "created": im.CreatedCount,
		"skipped": im.SkippedCount, "errors": im.ErrorCount, "message": im.Message,
	}
	if parsed && im.Meta != nil {
		m["parse"] = map[string]any{
			"headers": im.Meta["headers"], "preview_rows": im.Meta["preview_rows"],
			"total_rows": im.TotalRows, "detected_formats": im.Meta["detected_formats"],
			"suggested_mapping": im.Meta["suggested_mapping"],
		}
	} else {
		m["parse"] = nil
	}
	if im.Status == "completed" {
		m["result"] = map[string]any{
			"created": im.CreatedCount, "skipped_duplicates": im.SkippedCount,
			"errors": im.Errors, "created_currencies": im.Meta["created_currencies"],
			"created_tags": im.Meta["created_tags"], "created_categories": im.Meta["created_categories"],
		}
	} else {
		m["result"] = nil
	}
	return m
}

func Backup(b domain.Backup) map[string]any {
	schemaVersion, status := backupVersionStatus(b.AppVersion)
	return map[string]any{
		"filename": b.Filename, "size": b.Size, "note": b.Note,
		"schemaVersion": schemaVersion, "schemaStatus": status,
		"createdAt": func() any {
			if b.CreatedAt == nil {
				return nil
			}
			return b.CreatedAt.UTC().Format(time.RFC3339Nano)
		}(),
	}
}

func backupVersionStatus(appVersion *string) (any, string) {
	if appVersion == nil {
		return nil, "unknown"
	}
	v := *appVersion
	if v == "" || strings.Contains(v, "${") {
		return nil, "unknown"
	}
	if v == version.Value {
		return v, "current"
	}
	return v, "outdated"
}

func AccountDebt(a domain.Account) map[string]any {
	target := decimal.Zero
	if a.TargetAmount != nil {
		target = *a.TargetAmount
	}
	remaining := a.Balance
	progress := 0.0
	if target.IsPositive() {
		progress = target.Sub(remaining).Div(target).Mul(decimal.NewFromInt(100)).InexactFloat64()
		if progress < 0 {
			progress = 0
		}
	}
	dec := decimalsOf(&a)
	label := ""
	if a.DebtType != nil {
		switch *a.DebtType {
		case "i_owe":
			label = "I owe"
		case "owed_to_me":
			label = "Owed to me"
		}
	}
	m := Account(a)
	m["debtType"] = a.DebtType
	m["debtTypeLabel"] = label
	m["targetAmount"] = money.Number(target, dec)
	m["remainingDebt"] = money.Number(remaining, dec)
	m["paymentProgress"] = progress
	m["dueDate"] = a.DueDate
	m["counterparty"] = a.Counterparty
	m["description"] = a.DebtDesc
	m["isPaidOff"] = a.IsPaidOff
	return m
}

func WebAuthnCred(c auth.WebAuthnCred) map[string]any {
	return map[string]any{
		"id": c.ID, "name": c.Name, "aaguid": c.AAGUID,
		"last_used_at": c.LastUsedAt, "created_at": c.CreatedAt,
	}
}
