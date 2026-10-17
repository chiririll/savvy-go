package mcpserver

import (
	"context"
	"net/url"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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

type accountsArgs struct {
	SpaceID     int64 `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	WithSummary bool  `json:"with_summary,omitempty" jsonschema:"Also return totals across the accounts."`
}

type idArgs struct {
	SpaceID int64 `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID      int64 `json:"id" jsonschema:"Id of the item."`
}

type transactionsArgs struct {
	SpaceID       int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	StartDate     string  `json:"start_date,omitempty" jsonschema:"Earliest date, YYYY-MM-DD."`
	EndDate       string  `json:"end_date,omitempty" jsonschema:"Latest date, YYYY-MM-DD."`
	Type          string  `json:"type,omitempty" jsonschema:"income, expense or transfer."`
	Status        string  `json:"status,omitempty" jsonschema:"completed, pending or skipped."`
	AccountID     int64   `json:"account_id,omitempty" jsonschema:"Only this account."`
	CategoryIDs   []int64 `json:"category_ids,omitempty" jsonschema:"Only these categories."`
	TagIDs        []int64 `json:"tag_ids,omitempty" jsonschema:"Only these tags."`
	SortBy        string  `json:"sort_by,omitempty" jsonschema:"date (default), amount or created_at."`
	SortDirection string  `json:"sort_direction,omitempty" jsonschema:"desc (default) or asc."`
	Page          int     `json:"page,omitempty" jsonschema:"Page number from 1."`
	PerPage       int     `json:"per_page,omitempty" jsonschema:"Rows per page, at most 100 (default 25)."`
	WithSummary   bool    `json:"with_summary,omitempty" jsonschema:"Also return totals."`
}

type reportArgs struct {
	SpaceID     int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Report      string  `json:"report" jsonschema:"One of: overview, money_flow, expense_pace, expenses_by_category, cash_flow_over_time, transactions_summary, transactions_by_category, transactions_dynamics, transactions_top, net_worth, net_worth_history."`
	PeriodType  string  `json:"period_type,omitempty" jsonschema:"last_30_days (default), month, quarter, year, ytd or custom."`
	PeriodValue string  `json:"period_value,omitempty" jsonschema:"For month 2026-03, for quarter 2026-Q1, for year 2026."`
	StartDate   string  `json:"start_date,omitempty" jsonschema:"With period_type custom: first day, YYYY-MM-DD."`
	EndDate     string  `json:"end_date,omitempty" jsonschema:"With period_type custom: last day, YYYY-MM-DD."`
	CompareWith string  `json:"compare_with,omitempty" jsonschema:"none (default), previous_period or same_period_last_year."`
	AccountIDs  []int64 `json:"account_ids,omitempty" jsonschema:"Only these accounts."`
	CategoryIDs []int64 `json:"category_ids,omitempty" jsonschema:"Only these categories."`
	TagIDs      []int64 `json:"tag_ids,omitempty" jsonschema:"Only these tags."`
}

// mcpReadNeeds is the resource each read tool needs; list_spaces needs none.
var mcpReadNeeds = map[string]string{
	"list_accounts": "accounts", "list_categories": "categories", "list_tags": "tags", "list_currencies": "currencies",
	"list_budgets": "budgets", "list_debts": "debts", "get_debts_summary": "debts",
	"list_transactions": "transactions", "get_transaction": "transactions", "get_transactions_summary": "transactions",
	"get_report": "reports", "list_automation_rules": "automation", "get_automation_rule": "automation",
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
			func(ctx context.Context, req *mcp.CallToolRequest, in spaceArgs) (*mcp.CallToolResult, any, error) {
				return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, path), nil), nil, nil
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
			Name:        "list_accounts",
			Description: "List the space's accounts with their currency and current balance.",
			Annotations: ro("List accounts"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in accountsArgs) (*mcp.CallToolResult, any, error) {
			q := url.Values{}
			if in.WithSummary {
				q.Set("with_summary", "1")
			}
			return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/accounts"), q), nil, nil
		},
	)

	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name:        "list_transactions",
			Description: "List transactions, newest first, with optional filters. Each row has its own currency. Returns at most 100 rows per page; use page to continue.",
			Annotations: ro("List transactions"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in transactionsArgs) (*mcp.CallToolResult, any, error) {
			q := url.Values{}
			if err := firstErr(
				setDate(q, "start_date", in.StartDate), setDate(q, "end_date", in.EndDate),
				setEnum(q, "type", in.Type, "income", "expense", "transfer"),
				setEnum(q, "status", in.Status, "completed", "pending", "skipped"),
				setEnum(q, "sort_by", in.SortBy, "date", "amount", "created_at"),
				setEnum(q, "sort_direction", in.SortDirection, "asc", "desc"),
			); err != nil {
				return mcpFail(err.Error()), nil, nil
			}
			setID(q, "account_id", in.AccountID)
			setIDs(q, "category_ids", in.CategoryIDs)
			setIDs(q, "tag_ids", in.TagIDs)
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
			return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/transactions"), q), nil, nil
		},
	)

	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name:        "get_transaction",
			Description: "Get one transaction with its items, tags and category.",
			Annotations: ro("Get transaction"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in idArgs) (*mcp.CallToolResult, any, error) {
			return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/transactions/"+strconv.FormatInt(in.ID, 10)), nil), nil, nil
		},
	)

	mcp.AddTool(
		srv,
		&mcp.Tool{
			Name:        "get_report",
			Description: "Run one of the space's reports (spending by category, cash flow, net worth, ...) for a period. Totals are in the space's base currency unless the report says otherwise.",
			Annotations: ro("Get report"),
		},
		func(ctx context.Context, req *mcp.CallToolRequest, in reportArgs) (*mcp.CallToolResult, any, error) {
			route, ok := mcpReports[in.Report]
			if !ok {
				return mcpFail("Unknown report " + strconv.Quote(in.Report) + "."), nil, nil
			}
			q := url.Values{}
			if err := firstErr(
				setDate(q, "start_date", in.StartDate), setDate(q, "end_date", in.EndDate),
				setEnum(q, "period_type", in.PeriodType, "last_30_days", "month", "quarter", "year", "ytd", "custom"),
				setEnum(q, "compare_with", in.CompareWith, "none", "previous_period", "same_period_last_year"),
			); err != nil {
				return mcpFail(err.Error()), nil, nil
			}
			if in.PeriodValue != "" {
				q.Set("period_value", in.PeriodValue)
			}
			setIDs(q, "account_ids", in.AccountIDs)
			setIDs(q, "category_ids", in.CategoryIDs)
			setIDs(q, "tag_ids", in.TagIDs)
			return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/reports/"+route), q), nil, nil
		},
	)
}
