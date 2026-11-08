package mcpserver

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"savvy-go/internal/httpserver/httpx/dto"
)

// mcpReports maps get_report's report names to the REST report routes.
var mcpReports = map[string]string{
	"overview":                 "overview",
	"money_flow":               "money-flow",
	"expense_pace":             "expense-pace",
	"expenses_by_category":     "expenses-by-category",
	"cash_flow_over_time":      "cash-flow-over-time",
	"transactions_summary":     "transactions/summary",
	"transactions_by_category": "transactions/by-category",
	"transactions_dynamics":    "transactions/dynamics",
	"transactions_top":         "transactions/top",
	"net_worth":                "net-worth",
	"net_worth_history":        "net-worth-history",
}

type spaceArgs struct {
	SpaceID int64 `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
}

// listArgs is spaceArgs for the lists that can be trimmed with fields.
type listArgs struct {
	SpaceID int64    `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Fields  []string `json:"fields,omitempty" jsonschema:"Keys to keep of each item, to make the answer small, e.g. [id, name]. A dotted path picks inside a nested object or list: currency.code. Default all."`
}

// getArgs is idArgs for the one item that can be trimmed with fields.
type getArgs struct {
	SpaceID int64    `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID      int64    `json:"id" jsonschema:"Id of the item."`
	Fields  []string `json:"fields,omitempty" jsonschema:"Keys to keep of each item, to make the answer small, e.g. [id, name]. A dotted path picks inside a nested object or list: currency.code. Default all."`
}

type accountsArgs struct {
	SpaceID     int64    `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	WithSummary bool     `json:"with_summary,omitempty" jsonschema:"Also return totals across the accounts."`
	Type        string   `json:"type,omitempty" jsonschema:"Only these types, comma-separated: bank, crypto, cash, credit, debt, space."`
	Fields      []string `json:"fields,omitempty" jsonschema:"Keys to keep of each item, to make the answer small, e.g. [id, name]. A dotted path picks inside a nested object or list: currency.code. Default all."`
}

type balanceArgs struct {
	SpaceID    int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	AccountIDs []int64 `json:"account_ids,omitempty" jsonschema:"Only these accounts, from list_accounts. Default all active accounts, with a total."`
	At         string  `json:"at,omitempty" jsonschema:"Day, YYYY-MM-DD: the balance at the end of it. Default today."`
}

type balanceHistoryArgs struct {
	SpaceID    int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	AccountIDs []int64 `json:"account_ids,omitempty" jsonschema:"Only these accounts, from list_accounts. Default all active accounts, with a total."`
	From       string  `json:"from" jsonschema:"First day, YYYY-MM-DD."`
	To         string  `json:"to" jsonschema:"Last day, YYYY-MM-DD, at most about ten years after from."`
}

type idArgs struct {
	SpaceID int64 `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID      int64 `json:"id" jsonschema:"Id of the item."`
}

type transactionsArgs struct {
	SpaceID       int64    `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	StartDate     string   `json:"start_date,omitempty" jsonschema:"Earliest date, YYYY-MM-DD."`
	EndDate       string   `json:"end_date,omitempty" jsonschema:"Latest date, YYYY-MM-DD."`
	Type          string   `json:"type,omitempty" jsonschema:"income, expense or transfer."`
	Status        string   `json:"status,omitempty" jsonschema:"completed, pending or skipped."`
	AccountID     int64    `json:"account_id,omitempty" jsonschema:"Only this account."`
	AccountIDs    []int64  `json:"account_ids,omitempty" jsonschema:"Only these source accounts."`
	Currency      []string `json:"currency,omitempty" jsonschema:"Only these currency codes, e.g. USD. A transfer matches if either account is in one."`
	CategoryIDs   []int64  `json:"category_ids,omitempty" jsonschema:"Only these categories."`
	TagIDs        []int64  `json:"tag_ids,omitempty" jsonschema:"Only these tags."`
	Query         string   `json:"q,omitempty" jsonschema:"Search words that must all match description, item names or an amount (1879, 18.79)."`
	SortBy        string   `json:"sort_by,omitempty" jsonschema:"date (default), amount or created_at."`
	SortDirection string   `json:"sort_direction,omitempty" jsonschema:"desc (default) or asc."`
	Page          int      `json:"page,omitempty" jsonschema:"Page number from 1."`
	PerPage       int      `json:"per_page,omitempty" jsonschema:"Rows per page, at most 100 (default 25)."`
	WithSummary   bool     `json:"with_summary,omitempty" jsonschema:"Also return totals."`
	Fields        []string `json:"fields,omitempty" jsonschema:"Keys to keep in each row: id, type, description, date, status, amount, rowAmount, receivedAmount, payments, lines, tags, createdAt, ... A dotted path picks inside a list: payments.accountId, payments.amount, lines.categoryId. Default all, with the full account and category of every payment and line."`
}

// txFieldKeys are the keys of a transaction row, as the REST API names them.
var txFieldKeys = jsonKeys(dto.Transaction{})

type dayTotalsArgs struct {
	SpaceID     int64    `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Dates       []string `json:"dates,omitempty" jsonschema:"Only these days, YYYY-MM-DD. Give these, or both start_date and end_date."`
	StartDate   string   `json:"start_date,omitempty" jsonschema:"Earliest date, YYYY-MM-DD. With end_date."`
	EndDate     string   `json:"end_date,omitempty" jsonschema:"Latest date, YYYY-MM-DD. With start_date."`
	Type        string   `json:"type,omitempty" jsonschema:"income, expense or transfer."`
	AccountID   int64    `json:"account_id,omitempty" jsonschema:"Only this account."`
	AccountIDs  []int64  `json:"account_ids,omitempty" jsonschema:"Only these source accounts."`
	Currency    []string `json:"currency,omitempty" jsonschema:"Only these currency codes, e.g. USD."`
	CategoryIDs []int64  `json:"category_ids,omitempty" jsonschema:"Only these categories."`
	TagIDs      []int64  `json:"tag_ids,omitempty" jsonschema:"Only these tags."`
	Query       string   `json:"q,omitempty" jsonschema:"Search words that must all match description, item names or an amount."`
}

type reportArgs struct {
	SpaceID     int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Report      string  `json:"report" jsonschema:"Report name, one of those in the tool description."`
	PeriodType  string  `json:"period_type,omitempty" jsonschema:"Default last_30_days; month, quarter, year, ytd, custom"`
	PeriodValue string  `json:"period_value,omitempty" jsonschema:"For month 2026-03, for quarter 2026-Q1, for year 2026."`
	StartDate   string  `json:"start_date,omitempty" jsonschema:"With period_type custom: first day, YYYY-MM-DD."`
	EndDate     string  `json:"end_date,omitempty" jsonschema:"With period_type custom: last day, YYYY-MM-DD."`
	CompareWith string  `json:"compare_with,omitempty" jsonschema:"none (default), previous_period, same_period_last_year"`
	AccountIDs  []int64 `json:"account_ids,omitempty" jsonschema:"Only these accounts."`
	CategoryIDs []int64 `json:"category_ids,omitempty" jsonschema:"Only these categories."`
	TagIDs      []int64 `json:"tag_ids,omitempty" jsonschema:"Only these tags."`
}

// mcpReadNeeds is the resource each read tool needs; list_spaces needs none.
var mcpReadNeeds = map[string]string{
	"list_accounts": "accounts", "get_balance": "accounts", "get_balance_history": "accounts", "list_categories": "categories", "list_tags": "tags", "list_currencies": "currencies",
	"list_budgets": "budgets", "list_debts": "debts", "get_debts_summary": "debts",
	"list_transactions": "transactions", "get_transaction": "transactions", "get_transactions_summary": "transactions",
	"get_transactions_day_totals": "transactions",
	"get_report":                  "reports", "list_automation_rules": "automation", "get_automation_rule": "automation",
	"get_automation_rule_logs": "automation",
}

func ro(title string) *mcp.ToolAnnotations {
	no := false
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: &no}
}

// addSimpleReadTools adds the read tools that take no filters.
func (h Handler) addSimpleReadTools(srv *mcp.Server) {
	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name:        "list_spaces",
			Description: "List the spaces (separate ledgers) you can read, with your role in each.",
			Annotations: ro("List spaces"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			return h.mcpGet(ctx, req, "/api/spaces", nil), nil, nil
		},
	)

	spaceTool := func(name, title, desc, path string) {
		mcp.AddTool(srv, &mcp.Tool{Name: name, Description: desc, Annotations: ro(title)},
			func(ctx context.Context, req *mcp.CallToolRequest, in listArgs) (*mcp.CallToolResult, any, error) {
				return h.mcpGetFields(ctx, req, mcpSpacePath(in.SpaceID, path), nil, in.Fields, nil), nil, nil
			})
	}
	spaceTool("list_categories", "List categories", "List the space's categories (expense and income).", "/categories")
	spaceTool("list_tags", "List tags", "List the space's tags.", "/tags")
	spaceTool(
		"list_currencies",
		"List currencies",
		"List the currencies the space uses, with decimals, exchange rates and which one is the base currency.",
		"/currencies",
	)
	spaceTool("list_budgets", "List budgets", "List budgets with their progress.", "/budgets")
	spaceTool("list_debts", "List debts", "List debts: what is owed and to whom, with payment progress.", "/debts")
	spaceTool("get_debts_summary", "Debts summary", "Totals of the space's debts.", "/debts-summary")
	spaceTool(
		"get_transactions_summary",
		"Transactions summary",
		"Totals of the space's transactions.",
		"/transactions-summary",
	)

	spaceTool(
		"list_automation_rules",
		"List automation rules",
		"List the space's automation rules: what they match, what they do, whether they are active and how often they ran.",
		"/automation-rules",
	)
	for name, d := range map[string][3]string{
		"get_automation_rule":      {"Get automation rule", "Get one automation rule with its conditions and actions.", ""},
		"get_automation_rule_logs": {"Automation rule log", "What a rule did to which transactions, newest first.", "/logs"},
	} {
		mcp.AddTool(srv, &mcp.Tool{Name: name, Description: d[1], Annotations: ro(d[0])},
			func(ctx context.Context, req *mcp.CallToolRequest, in idArgs) (*mcp.CallToolResult, any, error) {
				return h.mcpGet(
					ctx,
					req,
					mcpSpacePath(in.SpaceID, "/automation-rules/"+strconv.FormatInt(in.ID, 10)+d[2]),
					nil,
				), nil, nil
			})
	}

}

// addQueryReadTools adds the read tools that turn their arguments into a query.
func (h Handler) addQueryReadTools(srv *mcp.Server) {
	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name: "list_accounts",
			Description: "List the space's accounts with their currency and current balance. A credit card (type credit) " +
				"also has its terms, available credit, last statement with the amount due and due date, and a status " +
				"(ok, warning or danger).",
			Annotations: ro("List accounts"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in accountsArgs) (*mcp.CallToolResult, any, error) {
			q := url.Values{}
			if in.Type != "" {
				q.Set("type", in.Type)
			}
			if in.WithSummary {
				q.Set("with_summary", "1")
			}
			return h.mcpGetFields(ctx, req, mcpSpacePath(in.SpaceID, "/accounts"), q, in.Fields, nil), nil, nil
		},
	)

	// Both read the balance chart's route, so a balance is computed in one place.
	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name: "get_balance",
			Description: "The balance of accounts at the end of a day (all active ones, or account_ids): opening balance plus " +
				"confirmed payments up to then. One series per account with one day, plus a total; native_data is in the account's own currency.",
			Annotations: ro("Get balance"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in balanceArgs) (*mcp.CallToolResult, any, error) {
			at := in.At
			if at == "" {
				at = time.Now().Format("2006-01-02")
			}
			q, err := balanceQuery(in.AccountIDs, at, at)
			if err != nil {
				return mcpFail(err.Error()), nil, nil
			}
			return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/accounts-balance-history"), q), nil, nil
		},
	)
	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name: "get_balance_history",
			Description: "The balance of accounts (all active ones, or account_ids) at the end of every day from from to to, " +
				"oldest first. One series per account, plus a total. " +
				"native_data is in the account's own currency, data in the space's base currency.",
			Annotations: ro("Get balance history"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in balanceHistoryArgs) (*mcp.CallToolResult, any, error) {
			if in.From == "" || in.To == "" {
				return mcpFail("give both from and to."), nil, nil
			}
			q, err := balanceQuery(in.AccountIDs, in.From, in.To)
			if err != nil {
				return mcpFail(err.Error()), nil, nil
			}
			return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/accounts-balance-history"), q), nil, nil
		},
	)

	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name: "list_transactions",
			Description: "List transactions, newest first, with optional filters. Each row has its own currency. Returns at " +
				"most 100 rows per page; use page to continue. Rows are large (every payment repeats its account): " +
				"pass fields, e.g. [\"id\",\"date\",\"description\",\"payments.accountId\",\"payments.amount\"], for a compact answer.",
			Annotations: ro("List transactions"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in transactionsArgs) (*mcp.CallToolResult, any, error) {
			q, err := transactionsQuery(in)
			if err != nil {
				return mcpFail(err.Error()), nil, nil
			}
			return h.mcpGetFields(ctx, req, mcpSpacePath(in.SpaceID, "/transactions"), q, in.Fields, txFieldKeys), nil, nil
		},
	)
	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name: "get_transactions_day_totals",
			Description: "What each day netted (income less expense; a transfer counts only its fee; skipped ones not at " +
				"all), per currency and largest first. Covers every transaction of the day, however long the list is. " +
				"Give dates, or start_date and end_date.",
			Annotations: ro("Transactions day totals"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in dayTotalsArgs) (*mcp.CallToolResult, any, error) {
			q, err := dayTotalsQuery(in)
			if err != nil {
				return mcpFail(err.Error()), nil, nil
			}
			return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/transactions-day-totals"), q), nil, nil
		},
	)
	h.addLookupReadTools(srv)
}

// addLookupReadTools adds the read tools that fetch one transaction or report.
func (h Handler) addLookupReadTools(srv *mcp.Server) {
	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name:        "get_transaction",
			Description: "Get one transaction with its items, tags and category.",
			Annotations: ro("Get transaction"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in getArgs) (*mcp.CallToolResult, any, error) {
			path := mcpSpacePath(in.SpaceID, "/transactions/"+strconv.FormatInt(in.ID, 10))
			return h.mcpGetFields(ctx, req, path, nil, in.Fields, txFieldKeys), nil, nil
		},
	)

	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name: "get_report",
			Description: "Run one of the space's reports (spending by category, cash flow, net worth, ...) for a period. " +
				"Totals are in the space's base currency unless the report says otherwise. Reports: overview, " +
				"money_flow, expense_pace, expenses_by_category, cash_flow_over_time, transactions_summary, " +
				"transactions_by_category, transactions_dynamics, transactions_top, net_worth, net_worth_history.",
			Annotations: ro("Get report"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in reportArgs) (*mcp.CallToolResult, any, error) {
			route, ok := mcpReports[in.Report]
			if !ok {
				return mcpFail("Unknown report " + strconv.Quote(in.Report) + "."), nil, nil
			}
			q, err := reportQuery(in)
			if err != nil {
				return mcpFail(err.Error()), nil, nil
			}
			return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/reports/"+route), q), nil, nil
		},
	)
}

// transactionsQuery turns the arguments of list_transactions into a query.
func transactionsQuery(in transactionsArgs) (url.Values, error) {
	q := url.Values{}
	if err := firstErr(
		setDate(q, "start_date", in.StartDate), setDate(q, "end_date", in.EndDate),
		setEnum(q, "type", in.Type, "income", "expense", "transfer"),
		setEnum(q, "status", in.Status, "completed", "pending", "skipped"),
		setEnum(q, "sort_by", in.SortBy, "date", "amount", "created_at"),
		setEnum(q, "sort_direction", in.SortDirection, "asc", "desc"),
	); err != nil {
		return nil, err
	}
	setID(q, "account_id", in.AccountID)
	setIDs(q, "account_ids", in.AccountIDs)
	setIDs(q, "category_ids", in.CategoryIDs)
	setIDs(q, "tag_ids", in.TagIDs)
	var codes []string
	for _, code := range in.Currency {
		if code = strings.TrimSpace(code); code != "" {
			codes = append(codes, code)
		}
	}
	if len(codes) > 0 {
		q.Set("currency", strings.Join(codes, ","))
	}
	if in.Query != "" {
		q.Set("q", in.Query)
	}
	if in.Page > 0 {
		q.Set("page", strconv.Itoa(in.Page))
	}
	per := in.PerPage
	if per <= 0 {
		per = 25
	}
	q.Set("per_page", strconv.Itoa(min(per, mcpMaxPerPage)))
	if in.WithSummary {
		q.Set("with_summary", "1")
	}
	return q, nil
}

// balanceQuery turns the arguments of the balance tools into a query.
func balanceQuery(accounts []int64, from, to string) (url.Values, error) {
	q := url.Values{}
	if err := firstErr(setDate(q, "start_date", from), setDate(q, "end_date", to)); err != nil {
		return nil, err
	}
	setIDs(q, "account_ids", accounts)
	return q, nil
}

// dayTotalsQuery turns the arguments of get_transactions_day_totals into a query.
func dayTotalsQuery(in dayTotalsArgs) (url.Values, error) {
	q, err := transactionsQuery(transactionsArgs{
		StartDate: in.StartDate, EndDate: in.EndDate, Type: in.Type, AccountID: in.AccountID,
		AccountIDs: in.AccountIDs, Currency: in.Currency, CategoryIDs: in.CategoryIDs, TagIDs: in.TagIDs, Query: in.Query,
	})
	if err != nil {
		return nil, err
	}
	q.Del("per_page")
	for _, d := range in.Dates {
		if err := setDate(url.Values{}, "dates", d); err != nil {
			return nil, err
		}
	}
	if len(in.Dates) > 0 {
		q.Set("dates", strings.Join(in.Dates, ","))
	}
	if len(in.Dates) == 0 && (in.StartDate == "" || in.EndDate == "") {
		return nil, errors.New("give dates, or both start_date and end_date.")
	}
	return q, nil
}

// reportQuery turns the arguments of get_report into a query.
func reportQuery(in reportArgs) (url.Values, error) {
	q := url.Values{}
	if err := firstErr(
		setDate(q, "start_date", in.StartDate), setDate(q, "end_date", in.EndDate),
		setEnum(q, "period_type", in.PeriodType, "last_30_days", "month", "quarter", "year", "ytd", "custom"),
		setEnum(q, "compare_with", in.CompareWith, "none", "previous_period", "same_period_last_year"),
	); err != nil {
		return nil, err
	}
	if in.PeriodValue != "" {
		q.Set("period_value", in.PeriodValue)
	}
	setIDs(q, "account_ids", in.AccountIDs)
	setIDs(q, "category_ids", in.CategoryIDs)
	setIDs(q, "tag_ids", in.TagIDs)
	return q, nil
}
