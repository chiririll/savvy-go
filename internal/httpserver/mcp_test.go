package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"savvy-go/internal/auth"
	"savvy-go/internal/auth/permissions"
)

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	if b.token != "" {
		r.Header.Set("Authorization", "Bearer "+b.token)
	}
	return http.DefaultTransport.RoundTrip(r)
}

// mcpSession connects the official MCP client to /mcp with a bearer token.
func (a *testApp) mcpSession(token string) *mcp.ClientSession {
	a.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	a.t.Cleanup(cancel)
	c := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	sess, err := c.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: a.srv.URL + "/mcp", HTTPClient: &http.Client{Transport: bearerTransport{token}},
	}, nil)
	if err != nil {
		a.t.Fatalf("connect: %v", err)
	}
	a.t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// mcpGrant connects with a token holding exactly these permissions in every space.
func (a *testApp) mcpGrant(u *auth.User, perms ...string) *mcp.ClientSession {
	a.t.Helper()
	tok, _ := a.issueGrant(u, auth.Grant{Permissions: perms, AllSpaces: true}, nil)
	return a.mcpSession(tok)
}

// mcpFull connects with everything an assistant can be given, as the admin
// preset of the consent screen does.
func (a *testApp) mcpFull(u *auth.User) *mcp.ClientSession {
	a.t.Helper()
	perms, err := permissions.FromPreset(permissions.PresetAdmin, true, permissions.MCPResources)
	if err != nil {
		a.t.Fatal(err)
	}
	return a.mcpGrant(u, perms...)
}

func (a *testApp) call(sess *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	a.t.Helper()
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		a.t.Fatalf("%s: %v", name, err)
	}
	return res
}

func (a *testApp) tools(sess *mcp.ClientSession) []*mcp.Tool {
	a.t.Helper()
	list, err := sess.ListTools(context.Background(), nil)
	if err != nil {
		a.t.Fatal(err)
	}
	return list.Tools
}

func toolNames(tools []*mcp.Tool) []string {
	var out []string
	for _, tl := range tools {
		out = append(out, tl.Name)
	}
	return out
}

func resultText(res *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range res.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			sb.WriteString(t.Text)
		}
	}
	return sb.String()
}

// toolData decodes what a tool returned: the REST body, so {"data": ...}.
func toolData(t *testing.T, res *mcp.CallToolResult) any {
	t.Helper()
	var body struct {
		Data any `json:"data"`
	}
	if err := json.Unmarshal([]byte(resultText(res)), &body); err != nil || body.Data == nil {
		t.Fatalf("no data in %s (%v)", resultText(res), err)
	}
	return body.Data
}

// idOf reads the id of the item a write tool returned.
func idOf(t *testing.T, res *mcp.CallToolResult) int64 {
	t.Helper()
	item, ok := toolData(t, res).(map[string]any)
	id, _ := item["id"].(float64)
	if !ok || id == 0 {
		t.Fatalf("no id in %s", resultText(res))
	}
	return int64(id)
}

func TestMCPRequiresABearerTokenAndPointsToTheMetadata(t *testing.T) {
	a := newTestApp(t)
	for _, auth := range []string{"", "Bearer nope", "Bearer svy_unknown"} {
		req, _ := http.NewRequest("POST", a.srv.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		res, err := a.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%q: status %d", auth, res.StatusCode)
		}
		if h := res.Header.Get("WWW-Authenticate"); !strings.Contains(h, `resource_metadata="http://localhost:8080/.well-known/oauth-protected-resource"`) {
			t.Fatalf("%q: WWW-Authenticate = %q", auth, h)
		}
	}
}

func TestMCPSessionCookieIsNotEnough(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)
	req, _ := http.NewRequest("POST", a.srv.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.AddCookie(&http.Cookie{Name: "svy_session", Value: sess.Token})
	res, err := a.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	status(t, res, http.StatusUnauthorized, "a browser session must not reach /mcp")
}

// A client decides what to run unasked from these hints. DestructiveHint is
// true when absent, so every tool that is not a delete must say false.
func TestMCPToolsAreMarkedByWhatTheyDo(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")

	tools := a.tools(a.mcpFull(u))
	for _, name := range []string{"list_spaces", "list_accounts", "list_transactions", "get_report", "create_transaction", "update_transaction", "delete_transaction"} {
		if !slices.Contains(toolNames(tools), name) {
			t.Errorf("missing tool %s", name)
		}
	}
	for _, tool := range tools {
		ann := tool.Annotations
		if ann == nil {
			t.Errorf("tool %s has no annotations", tool.Name)
			continue
		}
		reads := strings.HasPrefix(tool.Name, "list_") || strings.HasPrefix(tool.Name, "get_")
		deletes := strings.HasPrefix(tool.Name, "delete_")
		if ann.ReadOnlyHint != reads {
			t.Errorf("tool %s: read-only hint must be %v", tool.Name, reads)
		}
		if !reads && (ann.DestructiveHint == nil || *ann.DestructiveHint != deletes) {
			t.Errorf("tool %s: destructive hint must be explicitly %v, got %v", tool.Name, deletes, ann.DestructiveHint)
		}
		if ann.OpenWorldHint == nil || *ann.OpenWorldHint {
			t.Errorf("tool %s talks only to this app, so open-world must be false", tool.Name)
		}
		if tool.Description == "" {
			t.Errorf("tool %s has no description", tool.Name)
		}
	}
}

func TestMCPToolsReadTheCallersSpacesAndNothingElse(t *testing.T) {
	a := newTestApp(t)
	admin := a.createUser("admin@a.com", "secret1", "admin")
	adminSess := a.issue(admin, false)
	res := a.do("POST", "/api/accounts", map[string]any{"name": "Barclays", "type": "bank", "currency_code": "GBP", "initial_balance": 100}, adminSess.Token, adminSess.CSRF)
	status(t, res, http.StatusCreated, "seed account")

	member := a.createUser("viewer@a.com", "secret1", roleViewer)
	outsider := a.createUser("out@a.com", "secret1", "user")

	ms := a.mcpGrant(member, "accounts:read")
	spaces := toolData(t, a.call(ms, "list_spaces", nil)).([]any)
	if len(spaces) != 1 || spaces[0].(map[string]any)["name"] != "Test" {
		t.Fatalf("list_spaces = %v", spaces)
	}
	accs := toolData(t, a.call(ms, "list_accounts", map[string]any{"space_id": a.space.ID})).([]any)
	if len(accs) != 1 {
		t.Fatalf("list_accounts = %v", accs)
	}
	if acc := accs[0].(map[string]any); acc["name"] != "Barclays" || acc["currency"].(map[string]any)["code"] != "GBP" {
		t.Fatalf("account = %v", acc)
	}

	// A user outside the space gets a tool error and sees no space.
	os := a.mcpGrant(outsider, "accounts:read")
	denied := a.call(os, "list_accounts", map[string]any{"space_id": a.space.ID})
	if !denied.IsError || strings.Contains(resultText(denied), "Barclays") {
		t.Fatalf("outsider = %v %s", denied.IsError, resultText(denied))
	}
	if spaces := toolData(t, a.call(os, "list_spaces", nil)).([]any); len(spaces) != 0 {
		t.Fatalf("outsider sees spaces: %v", spaces)
	}
}

func TestMCPListTransactionsForwardsFiltersAndBoundsThePage(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	s := a.mcpFull(u)

	acc := idOf(t, a.call(s, "create_account", map[string]any{"space_id": sp, "name": "Wallet", "type": "cash", "currency_code": "EUR", "initial_balance": "0"}))
	for _, tx := range []struct{ typ, desc, date string }{
		{"expense", "coffee", "2026-02-01"},
		{"income", "salary", "2026-02-02"},
		{"expense", "old rent", "2025-12-31"},
	} {
		res := a.call(s, "create_transaction", map[string]any{"space_id": sp, "type": tx.typ, "account_id": acc, "amount": "5", "description": tx.desc, "date": tx.date})
		if res.IsError {
			t.Fatalf("seed %s: %s", tx.desc, resultText(res))
		}
	}

	res := a.call(s, "list_transactions", map[string]any{"space_id": sp, "type": "expense", "start_date": "2026-01-01"})
	if res.IsError {
		t.Fatalf("list_transactions: %s", resultText(res))
	}
	list := toolData(t, res).([]any)
	if len(list) != 1 || list[0].(map[string]any)["description"] != "coffee" {
		t.Fatalf("type and start_date must both narrow the list: %s", resultText(res))
	}

	// The page size is capped, not passed through to the database.
	res = a.call(s, "list_transactions", map[string]any{"space_id": sp, "per_page": 5000})
	if res.IsError || !strings.Contains(resultText(res), `"per_page":100`) {
		t.Fatalf("page size must be capped at 100: %v %s", res.IsError, resultText(res))
	}

	// Input is validated before it can become part of a path or query.
	for name, args := range map[string]map[string]any{
		"date": {"space_id": sp, "start_date": "yesterday"},
		"type": {"space_id": sp, "type": "expense&per_page=1"},
	} {
		if res := a.call(s, "list_transactions", args); !res.IsError {
			t.Errorf("a bad %s must be refused: %s", name, resultText(res))
		}
	}
	if res := a.call(s, "get_report", map[string]any{"space_id": sp, "report": "../../auth/api-tokens"}); !res.IsError {
		t.Fatalf("an unknown report must be refused: %s", resultText(res))
	}
}

func TestMCPAcceptsOAuthTokenButStaysReadOnly(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	_, access := a.connect(a.issue(u, false))

	ms := a.mcpSession(access)
	if res := a.call(ms, "list_spaces", nil); res.IsError {
		t.Fatalf("list_spaces: %s", resultText(res))
	}
	res := a.call(ms, "create_tag", map[string]any{"space_id": a.space.ID, "name": "x"})
	if !res.IsError || !strings.Contains(resultText(res), "tags:write") {
		t.Fatalf("a connection approved without write access wrote: %v %s", res.IsError, resultText(res))
	}
}

// What a connection is given (permissions.MCPResources) must cover every read tool, or
// a tool would answer 403 for every connected assistant; a token with less than
// that gets an error naming the missing permission.
func TestMCPReadToolsNeedTheirPermissions(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID

	gs := a.mcpGrant(u, permissions.ReadAll(permissions.MCPResources...)...)
	calls := map[string]map[string]any{
		"list_spaces":              nil,
		"list_accounts":            {"space_id": sp},
		"list_categories":          {"space_id": sp},
		"list_tags":                {"space_id": sp},
		"list_currencies":          {"space_id": sp},
		"list_budgets":             {"space_id": sp},
		"list_debts":               {"space_id": sp},
		"list_automation_rules":    {"space_id": sp},
		"get_debts_summary":        {"space_id": sp},
		"get_transactions_summary": {"space_id": sp},
		"list_transactions":        {"space_id": sp},
		"get_report":               {"space_id": sp, "report": "overview"},
	}
	for name, args := range calls {
		if res := a.call(gs, name, args); res.IsError {
			t.Errorf("%s is refused to a connection: %s", name, resultText(res))
		}
	}
	// A new read tool must be added to calls above, or to the ones that need an id.
	needsID := []string{"get_transaction", "get_automation_rule", "get_automation_rule_logs"}
	for _, tool := range a.tools(gs) {
		if !tool.Annotations.ReadOnlyHint {
			t.Errorf("a read grant is offered the write tool %s", tool.Name)
		}
		if _, ok := calls[tool.Name]; !ok && !slices.Contains(needsID, tool.Name) {
			t.Errorf("read tool %s is not exercised by this test", tool.Name)
		}
	}

	ns := a.mcpGrant(u, "accounts:read")
	if res := a.call(ns, "list_accounts", map[string]any{"space_id": sp}); res.IsError {
		t.Fatalf("list_accounts: %s", resultText(res))
	}
	res := a.call(ns, "list_transactions", map[string]any{"space_id": sp})
	if !res.IsError || !strings.Contains(resultText(res), "transactions:read") {
		t.Fatalf("list_transactions with accounts:read = %v %s", res.IsError, resultText(res))
	}
}

// The write tools are plain REST calls made with the caller's token, so a token
// needs the grant of the route's level: write, and delete on top for deleting.
func TestMCPWriteToolsNeedWriteAndDeleteGrants(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID

	res := a.call(a.mcpGrant(u, "tags:read"), "create_tag", map[string]any{"space_id": sp, "name": "food"})
	if !res.IsError || !strings.Contains(resultText(res), "tags:write") {
		t.Fatalf("create_tag with tags:read = %v %s", res.IsError, resultText(res))
	}

	writer := a.mcpGrant(u, "tags:write")
	res = a.call(writer, "create_tag", map[string]any{"space_id": sp, "name": "food"})
	if res.IsError {
		t.Fatalf("create_tag: %s", resultText(res))
	}
	tag := idOf(t, res)
	res = a.call(writer, "delete_tag", map[string]any{"space_id": sp, "id": tag})
	if !res.IsError || !strings.Contains(resultText(res), "tags:delete") {
		t.Fatalf("delete_tag with tags:write = %v %s", res.IsError, resultText(res))
	}
	if !strings.Contains(resultText(a.call(writer, "list_tags", map[string]any{"space_id": sp})), `"food"`) {
		t.Fatal("a refused delete removed the tag")
	}

	res = a.call(a.mcpGrant(u, "tags:delete"), "delete_tag", map[string]any{"space_id": sp, "id": tag})
	if res.IsError {
		t.Fatalf("delete_tag: %s", resultText(res))
	}
	if strings.Contains(resultText(a.call(writer, "list_tags", map[string]any{"space_id": sp})), `"food"`) {
		t.Fatal("the tag is still there")
	}
}

func TestMCPWriteToolsRecordAndChangeMoney(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	s := a.mcpFull(u)
	balance := func() float64 {
		t.Helper()
		accs := toolData(t, a.call(s, "list_accounts", map[string]any{"space_id": sp})).([]any)
		return accs[0].(map[string]any)["currentBalance"].(float64)
	}

	acc := idOf(t, a.call(s, "create_account", map[string]any{"space_id": sp, "name": "Wallet", "type": "cash", "currency_code": "EUR", "initial_balance": "100.00"}))

	// Amounts travel as exact decimal strings and come back exact.
	res := a.call(s, "create_transaction", map[string]any{"space_id": sp, "type": "expense", "account_id": acc, "amount": "12.34", "description": "lunch", "date": "2026-03-01"})
	if res.IsError {
		t.Fatalf("create_transaction: %s", resultText(res))
	}
	tx := idOf(t, res)
	if got := toolData(t, res).(map[string]any)["amount"]; got != 12.34 {
		t.Fatalf("amount = %v, want 12.34", got)
	}
	if got := balance(); got != 87.66 {
		t.Fatalf("balance = %v, want 87.66", got)
	}

	res = a.call(s, "update_transaction", map[string]any{"space_id": sp, "id": tx, "amount": "20.00"})
	if res.IsError {
		t.Fatalf("update_transaction: %s", resultText(res))
	}
	if item := toolData(t, res).(map[string]any); item["amount"] != float64(20) || item["description"] != "lunch" || item["date"] != "2026-03-01" {
		t.Fatalf("update_transaction must change only the amount: %v", item)
	}
	if got := balance(); got != 80 {
		t.Fatalf("balance = %v, want 80", got)
	}

	if bad := a.call(s, "create_transaction", map[string]any{"space_id": sp, "type": "expense", "account_id": 0, "amount": "1"}); !bad.IsError {
		t.Fatalf("a transaction without an account must be refused: %s", resultText(bad))
	}
	if got := balance(); got != 80 {
		t.Fatalf("a refused transaction changed the balance to %v", got)
	}

	if res = a.call(s, "delete_transaction", map[string]any{"space_id": sp, "id": tx}); res.IsError {
		t.Fatalf("delete_transaction: %s", resultText(res))
	}
	if res = a.call(s, "get_transaction", map[string]any{"space_id": sp, "id": tx}); !res.IsError {
		t.Fatalf("the transaction is still there: %s", resultText(res))
	}
	if got := balance(); got != 100 {
		t.Fatalf("deleting must give the money back: balance %v", got)
	}
}

// An assistant is offered only the tools its token can use, so it does not try
// what would fail. Calling one anyway still gets the REST error.
func TestMCPListsOnlyTheToolsTheTokenCanUse(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	names := func(perms ...string) []string { return toolNames(a.tools(a.mcpGrant(u, perms...))) }

	viewer := names("accounts:read", "transactions:read")
	for _, want := range []string{"list_spaces", "list_accounts", "get_transaction"} {
		if !slices.Contains(viewer, want) {
			t.Errorf("a reader is missing %s: %v", want, viewer)
		}
	}
	for _, hidden := range []string{"list_tags", "get_report", "create_transaction", "delete_transaction", "create_account"} {
		if slices.Contains(viewer, hidden) {
			t.Errorf("%s is offered to a token that cannot use it: %v", hidden, viewer)
		}
	}

	editor := names("accounts:read", "transactions:write")
	for _, want := range []string{"create_transaction", "update_transaction"} {
		if !slices.Contains(editor, want) {
			t.Errorf("a writer is missing %s: %v", want, editor)
		}
	}
	for _, hidden := range []string{"delete_transaction", "create_account"} {
		if slices.Contains(editor, hidden) {
			t.Errorf("%s is offered without its grant: %v", hidden, editor)
		}
	}
	if all := names("transactions:delete"); !slices.Contains(all, "delete_transaction") {
		t.Errorf("delete is missing: %v", all)
	}

	// A stale client may still call a tool it was not offered and gets the REST reason.
	res := a.call(a.mcpGrant(u, "accounts:read"), "create_account", map[string]any{"space_id": a.space.ID, "name": "X", "type": "cash", "currency_code": "EUR"})
	if !res.IsError || !strings.Contains(resultText(res), "accounts:write") {
		t.Errorf("unoffered tool = %v %s", res.IsError, resultText(res))
	}
}

func TestMCPAutomationRules(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	s := a.mcpFull(u)

	res := a.call(s, "create_automation_rule", map[string]any{
		"space_id": sp, "name": "Coffee", "trigger_type": "on_transaction_create",
		"conditions": map[string]any{"match": "all", "conditions": []any{map[string]any{"field": "description", "op": "contains", "value": "coffee"}}},
		"actions":    []any{map[string]any{"type": "set_description", "value": "Coffee"}},
	})
	if res.IsError {
		t.Fatalf("create_automation_rule: %s", resultText(res))
	}
	id := idOf(t, res)

	list := toolData(t, a.call(s, "list_automation_rules", map[string]any{"space_id": sp})).([]any)
	if len(list) != 1 || list[0].(map[string]any)["name"] != "Coffee" {
		t.Fatalf("list_automation_rules = %v", list)
	}
	res = a.call(s, "toggle_automation_rule", map[string]any{"space_id": sp, "id": id})
	if res.IsError || toolData(t, res).(map[string]any)["isActive"] != false {
		t.Fatalf("toggle_automation_rule must switch an active rule off: %v %s", res.IsError, resultText(res))
	}
	if res = a.call(s, "get_automation_rule_logs", map[string]any{"space_id": sp, "id": id}); res.IsError {
		t.Fatalf("get_automation_rule_logs: %s", resultText(res))
	}
	if res = a.call(s, "delete_automation_rule", map[string]any{"space_id": sp, "id": id}); res.IsError {
		t.Fatalf("delete_automation_rule: %s", resultText(res))
	}
	if list := toolData(t, a.call(s, "list_automation_rules", map[string]any{"space_id": sp})).([]any); len(list) != 0 {
		t.Fatalf("the rule is still there: %v", list)
	}

	// Rules are their own part of a space: tokens for transactions do not see them.
	for _, name := range toolNames(a.tools(a.mcpGrant(u, "transactions:write"))) {
		if strings.Contains(name, "automation") {
			t.Errorf("%s is offered without automation access", name)
		}
	}
}
