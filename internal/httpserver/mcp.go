package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	savvyauth "savvy-go/internal/auth"
	"savvy-go/internal/version"
)

// The MCP server: tools for AI assistants, served over Streamable HTTP at
// /mcp. It has no data access of its own. Every tool calls one of the app's own
// routes in-process with the caller's bearer token, so roles, token
// permissions, token spaces, membership and the JSON shape (money, currencies) are exactly
// those of the REST API and cannot drift from it. The read tools are here, the
// write tools in mcp_write.go; a token that was only given read access can
// call a write tool and gets the REST 403 naming the missing permission.

const (
	mcpMaxResult    = 256 << 10
	mcpMaxPerPage   = 100
	mcpInstructions = "Go Savvy is a multi-currency expense tracker. Data lives in spaces: call list_spaces first and pass a space's id as space_id. " +
		"Amounts are exact decimal numbers in the currency of the account or transaction they belong to; never add amounts of different currencies, " +
		"and always say which currency a figure is in. Dates are YYYY-MM-DD. " +
		"Tools that create, change or delete data work only when the user granted write access; they act as the user, so do not use them unless the user asked for the change, " +
		"and ask before deleting. Amounts are strings, for example \"12.50\"."
)

var dateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

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

// mcpHandler serves /mcp. The OAuth metadata URL is read per request because
// the public URL can change in the admin panel.
func (s *Server) mcpHandler() http.Handler {
	h := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		var perms []string
		if info := auth.TokenInfoFromContext(r.Context()); info != nil {
			perms = info.Scopes
		}
		return s.mcpServer(perms)
	}, &mcp.StreamableHTTPOptions{
		// Stateless: nothing to lose on restart, no session to hijack.
		Stateless:    true,
		JSONResponse: true,
		// The check guards unauthenticated localhost servers against DNS
		// rebinding. Here every request needs a bearer token a rebinding page
		// cannot send, and behind a reverse proxy it would reject the public Host.
		DisableLocalhostProtection: true,
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		oauthCORS(w)
		opts := &auth.RequireBearerTokenOptions{
			ResourceMetadataURL:    s.appURL(r.Context()) + "/.well-known/oauth-protected-resource",
			AllowMissingExpiration: true,
		}
		auth.RequireBearerToken(s.verifyMCPToken, opts)(h).ServeHTTP(w, r)
	})
}

// verifyMCPToken accepts any live API token: OAuth access tokens and tokens
// made by hand in Settings alike. A browser session is not enough. What the
// token may read is decided per call, by the REST routes the tools use.
func (s *Server) verifyMCPToken(ctx context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
	t, err := s.apiTokens.Resolve(ctx, raw)
	if err != nil {
		return nil, err
	}
	if t == nil || t.User == nil {
		return nil, fmt.Errorf("%w: unknown or expired token", auth.ErrInvalidToken)
	}
	info := &auth.TokenInfo{Scopes: t.Permissions, UserID: strconv.FormatInt(t.UserID, 10)}
	if t.ExpiresAt != nil {
		info.Expiration = *t.ExpiresAt
	}
	return info, nil
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

// mcpNeed is the grant a tool's REST route asks of a token.
type mcpNeed struct{ resource, level string }

// mcpReadNeeds is the resource each read tool needs; list_spaces needs none.
var mcpReadNeeds = map[string]string{
	"list_accounts": "accounts", "list_categories": "categories", "list_tags": "tags", "list_currencies": "currencies",
	"list_budgets": "budgets", "list_debts": "debts", "get_debts_summary": "debts",
	"list_transactions": "transactions", "get_transaction": "transactions", "get_transactions_summary": "transactions",
	"get_report": "reports", "list_automation_rules": "automation", "get_automation_rule": "automation",
	"get_automation_rule_logs": "automation",
}

// mcpServer builds the tool set for a token holding perms. It is cheap and
// holds nothing per user: each call reads the caller's token from its own
// request. Tools the token has no grant for are left out of the list, so an
// assistant is not offered what would only fail. This is only a courtesy: a
// call to one anyway still reaches the REST route, which checks the grant and
// names the missing permission.
func (s *Server) mcpServer(perms []string) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: "go-savvy", Version: version.Value}, &mcp.ServerOptions{Instructions: mcpInstructions})
	s.addSimpleReadTools(srv)
	s.addQueryReadTools(srv)
	s.hideUngrantedTools(srv, s.addMCPWriteTools(srv), perms)
	return srv
}

func ro(title string) *mcp.ToolAnnotations {
	no := false
	return &mcp.ToolAnnotations{Title: title, ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: &no}
}

// addSimpleReadTools adds the read tools that take no filters.
func (s *Server) addSimpleReadTools(srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{Name: "list_spaces", Description: "List the spaces (separate ledgers) you can read, with your role in each.", Annotations: ro("List spaces")},
		func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			return s.mcpGet(ctx, req, "/api/spaces", nil), nil, nil
		})

	spaceTool := func(name, title, desc, path string) {
		mcp.AddTool(srv, &mcp.Tool{Name: name, Description: desc, Annotations: ro(title)},
			func(ctx context.Context, req *mcp.CallToolRequest, in spaceArgs) (*mcp.CallToolResult, any, error) {
				return s.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, path), nil), nil, nil
			})
	}
	spaceTool("list_categories", "List categories", "List the space's categories (expense and income).", "/categories")
	spaceTool("list_tags", "List tags", "List the space's tags.", "/tags")
	spaceTool("list_currencies", "List currencies", "List the currencies the space uses, with decimals, exchange rates and which one is the base currency.", "/currencies")
	spaceTool("list_budgets", "List budgets", "List budgets with their progress.", "/budgets")
	spaceTool("list_debts", "List debts", "List debts: what is owed and to whom, with payment progress.", "/debts")
	spaceTool("get_debts_summary", "Debts summary", "Totals of the space's debts.", "/debts-summary")
	spaceTool("get_transactions_summary", "Transactions summary", "Totals of the space's transactions.", "/transactions-summary")

	spaceTool("list_automation_rules", "List automation rules", "List the space's automation rules: what they match, what they do, whether they are active and how often they ran.", "/automation-rules")
	for name, d := range map[string][3]string{
		"get_automation_rule":      {"Get automation rule", "Get one automation rule with its conditions and actions.", ""},
		"get_automation_rule_logs": {"Automation rule log", "What a rule did to which transactions, newest first.", "/logs"},
	} {
		mcp.AddTool(srv, &mcp.Tool{Name: name, Description: d[1], Annotations: ro(d[0])},
			func(ctx context.Context, req *mcp.CallToolRequest, in idArgs) (*mcp.CallToolResult, any, error) {
				return s.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/automation-rules/"+strconv.FormatInt(in.ID, 10)+d[2]), nil), nil, nil
			})
	}

}

// addQueryReadTools adds the read tools that turn their arguments into a query.
func (s *Server) addQueryReadTools(srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{Name: "list_accounts", Description: "List the space's accounts with their currency and current balance.", Annotations: ro("List accounts")},
		func(ctx context.Context, req *mcp.CallToolRequest, in accountsArgs) (*mcp.CallToolResult, any, error) {
			q := url.Values{}
			if in.WithSummary {
				q.Set("with_summary", "1")
			}
			return s.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/accounts"), q), nil, nil
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "list_transactions", Description: "List transactions, newest first, with optional filters. Each row has its own currency. Returns at most 100 rows per page; use page to continue.", Annotations: ro("List transactions")},
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
			return s.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/transactions"), q), nil, nil
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "get_transaction", Description: "Get one transaction with its items, tags and category.", Annotations: ro("Get transaction")},
		func(ctx context.Context, req *mcp.CallToolRequest, in idArgs) (*mcp.CallToolResult, any, error) {
			return s.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/transactions/"+strconv.FormatInt(in.ID, 10)), nil), nil, nil
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "get_report", Description: "Run one of the space's reports (spending by category, cash flow, net worth, ...) for a period. Totals are in the space's base currency unless the report says otherwise.", Annotations: ro("Get report")},
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
			return s.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/reports/"+route), q), nil, nil
		})
}

// hideUngrantedTools leaves out of the tool list what the token has no grant for.
func (s *Server) hideUngrantedTools(srv *mcp.Server, needs map[string]mcpNeed, perms []string) {
	for name, resource := range mcpReadNeeds {
		needs[name] = mcpNeed{resource, savvyauth.PermRead}
	}
	token := &savvyauth.APIToken{Permissions: perms}
	srv.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			res, err := next(ctx, method, req)
			if list, ok := res.(*mcp.ListToolsResult); ok && err == nil {
				list.Tools = slices.DeleteFunc(list.Tools, func(t *mcp.Tool) bool {
					n, known := needs[t.Name]
					return known && !token.Allows(n.resource, n.level)
				})
			}
			return res, err
		}
	})
}

func mcpSpacePath(id int64, rest string) string {
	return "/api/spaces/" + strconv.FormatInt(id, 10) + rest
}

func mcpFail(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: msg}}}
}

func firstErr(errs ...error) error {
	for _, e := range errs {
		if e != nil {
			return e
		}
	}
	return nil
}

func setDate(q url.Values, key, v string) error {
	if v == "" {
		return nil
	}
	if !dateRe.MatchString(v) {
		return fmt.Errorf("%s must be YYYY-MM-DD.", key)
	}
	q.Set(key, v)
	return nil
}

func setEnum(q url.Values, key, v string, allowed ...string) error {
	if v == "" {
		return nil
	}
	for _, a := range allowed {
		if v == a {
			q.Set(key, v)
			return nil
		}
	}
	return fmt.Errorf("%s must be one of: %s.", key, strings.Join(allowed, ", "))
}

func setID(q url.Values, key string, id int64) {
	if id > 0 {
		q.Set(key, strconv.FormatInt(id, 10))
	}
}

func setIDs(q url.Values, key string, ids []int64) {
	var parts []string
	for _, id := range ids {
		if id > 0 {
			parts = append(parts, strconv.FormatInt(id, 10))
		}
	}
	if len(parts) > 0 {
		q.Set(key, strings.Join(parts, ","))
	}
}

// mcpGet runs a GET against the app's own router as the caller. The bearer
// token comes from the caller's /mcp request, so the usual middleware decides
// what they may see.
func (s *Server) mcpGet(ctx context.Context, req *mcp.CallToolRequest, path string, q url.Values) *mcp.CallToolResult {
	return s.mcpCall(ctx, req, http.MethodGet, path, q, nil)
}

// mcpCall is mcpGet for any method. A body is sent as JSON.
func (s *Server) mcpCall(ctx context.Context, req *mcp.CallToolRequest, method, path string, q url.Values, body []byte) *mcp.CallToolResult {
	var authz string
	if extra := req.GetExtra(); extra != nil {
		authz = extra.Header.Get("Authorization")
	}
	if authz == "" {
		return mcpFail("Unauthenticated.")
	}
	target := path
	if len(q) > 0 {
		target += "?" + q.Encode()
	}
	// chi would reuse the routing state of the /mcp request (method POST) for a
	// context that carries it, so the call starts with a clean one.
	ctx = context.WithValue(ctx, chi.RouteCtxKey, (*chi.Context)(nil))
	hr, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return mcpFail(err.Error())
	}
	hr.Header.Set("Authorization", authz)
	hr.Header.Set("Accept", "application/json")
	if body != nil {
		hr.Header.Set("Content-Type", "application/json")
	}
	rec := &memoryResponse{header: http.Header{}, status: http.StatusOK}
	s.mux.ServeHTTP(rec, hr)

	out := bytes.TrimSpace(rec.body.Bytes())
	if rec.status >= 400 {
		return mcpFail(apiErrorMessage(rec.status, out))
	}
	if len(out) == 0 { // 204: a delete has nothing to return
		out = []byte(`{"ok":true}`)
	}
	if len(out) > mcpMaxResult {
		return mcpFail("The result is too large. Narrow it with filters (dates, account, category) or ask for a smaller page.")
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(out)}}}
}

// apiErrorMessage is the message of a REST error body, or its status.
func apiErrorMessage(status int, body []byte) string {
	var e struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &e) == nil && e.Message != "" {
		return e.Message
	}
	return http.StatusText(status)
}

// memoryResponse is a ResponseWriter that keeps the answer in memory.
type memoryResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (m *memoryResponse) Header() http.Header         { return m.header }
func (m *memoryResponse) WriteHeader(status int)      { m.status = status }
func (m *memoryResponse) Write(b []byte) (int, error) { return m.body.Write(b) }
