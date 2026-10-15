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

func (a *testApp) call(sess *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	a.t.Helper()
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		a.t.Fatalf("%s: %v", name, err)
	}
	return res
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
	req.AddCookie(&http.Cookie{Name: "svy_session", Value: sess.Token})
	res, _ := a.client.Do(req)
	status(t, res, http.StatusUnauthorized, "a browser session must not reach /mcp")
}

func TestMCPToolsAreMarkedByWhatTheyDo(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", roleEditor)
	raw, _ := a.issueToken(u, "read-write", nil)
	sess := a.mcpSession(raw)

	tools, err := sess.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"list_spaces": false, "list_accounts": false, "list_transactions": false, "get_report": false}
	for _, tool := range tools.Tools {
		// A list_ or get_ tool never changes anything; every other tool does,
		// so no client may treat it as safe to run unasked.
		reads := strings.HasPrefix(tool.Name, "list_") || strings.HasPrefix(tool.Name, "get_")
		if tool.Annotations == nil || tool.Annotations.ReadOnlyHint != reads {
			t.Errorf("tool %s: read-only hint must be %v", tool.Name, reads)
		}
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
		if tool.Description == "" {
			t.Errorf("tool %s has no description", tool.Name)
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("missing tool %s", name)
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
	mTok, _ := a.issueToken(member, "read", nil)
	oTok, _ := a.issueToken(outsider, "read", nil)
	spaceID := a.space.ID

	ms := a.mcpSession(mTok)
	spaces := a.call(ms, "list_spaces", nil)
	if spaces.IsError || !strings.Contains(resultText(spaces), `"Test"`) {
		t.Fatalf("list_spaces = %v %s", spaces.IsError, resultText(spaces))
	}
	accs := a.call(ms, "list_accounts", map[string]any{"space_id": spaceID})
	if accs.IsError || !strings.Contains(resultText(accs), "Barclays") || !strings.Contains(resultText(accs), "GBP") {
		t.Fatalf("list_accounts = %v %s", accs.IsError, resultText(accs))
	}

	// A user who is not a member sees a clean tool error, not the data.
	os := a.mcpSession(oTok)
	denied := a.call(os, "list_accounts", map[string]any{"space_id": spaceID})
	if !denied.IsError || strings.Contains(resultText(denied), "Barclays") {
		t.Fatalf("outsider = %v %s", denied.IsError, resultText(denied))
	}
	none := a.call(os, "list_spaces", nil)
	if strings.Contains(resultText(none), `"Test"`) {
		t.Fatalf("outsider sees the space: %s", resultText(none))
	}
}

func TestMCPForwardsFiltersAndBoundsThePage(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	tok, _ := a.issueToken(u, "read", nil)
	sess := a.mcpSession(tok)

	res := a.call(sess, "list_transactions", map[string]any{"space_id": a.space.ID, "per_page": 5000, "type": "expense", "start_date": "2026-01-01"})
	if res.IsError {
		t.Fatalf("list_transactions: %s", resultText(res))
	}
	if !strings.Contains(resultText(res), `"per_page":100`) {
		t.Fatalf("page size must be capped at 100: %s", resultText(res))
	}

	// Inputs are validated before they can become part of a path or query.
	bad := a.call(sess, "get_report", map[string]any{"space_id": a.space.ID, "report": "../../auth/api-tokens"})
	if !bad.IsError {
		t.Fatalf("an unknown report must be refused: %s", resultText(bad))
	}
}

func TestMCPAcceptsTheOAuthAccessToken(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sess := a.issue(u, false)
	_, access := a.connect(sess)

	ms := a.mcpSession(access)
	if res := a.call(ms, "list_spaces", nil); res.IsError {
		t.Fatalf("list_spaces: %s", resultText(res))
	}
}

// What a connection is given (auth.MCPResources) must cover every tool, or a
// tool would answer 403 for every connected assistant; a token with less than
// that gets an error naming the missing permission.
func TestMCPToolsNeedTheirPermissions(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID

	granted, _ := a.issueGrant(u, auth.Grant{Permissions: auth.ReadPerms(auth.MCPResources...), AllSpaces: true}, nil)
	gs := a.mcpSession(granted)
	calls := map[string]map[string]any{
		"list_spaces":              nil,
		"list_accounts":            {"space_id": sp},
		"list_categories":          {"space_id": sp},
		"list_tags":                {"space_id": sp},
		"list_currencies":          {"space_id": sp},
		"list_budgets":             {"space_id": sp},
		"list_debts":               {"space_id": sp},
		"get_debts_summary":        {"space_id": sp},
		"get_transactions_summary": {"space_id": sp},
		"list_transactions":        {"space_id": sp},
		"get_report":               {"space_id": sp, "report": "overview"},
	}
	for name, args := range calls {
		if res := a.call(gs, name, args); res.IsError && strings.Contains(resultText(res), "permission") {
			t.Errorf("%s is refused to a connection: %s", name, resultText(res))
		}
	}

	narrow, _ := a.issueGrant(u, auth.Grant{Permissions: []string{"accounts:read"}, AllSpaces: true}, nil)
	ns := a.mcpSession(narrow)
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
	session := func(perms ...string) *mcp.ClientSession {
		tok, _ := a.issueGrant(u, auth.Grant{Permissions: perms, AllSpaces: true}, nil)
		return a.mcpSession(tok)
	}

	reader := session("tags:read")
	res := a.call(reader, "create_tag", map[string]any{"space_id": sp, "name": "food"})
	if !res.IsError || !strings.Contains(resultText(res), "tags:write") {
		t.Fatalf("create_tag with tags:read = %v %s", res.IsError, resultText(res))
	}

	writer := session("tags:write")
	res = a.call(writer, "create_tag", map[string]any{"space_id": sp, "name": "food"})
	if res.IsError {
		t.Fatalf("create_tag: %s", resultText(res))
	}
	tag := idOf(t, res)
	res = a.call(writer, "list_tags", map[string]any{"space_id": sp})
	if !strings.Contains(resultText(res), `"food"`) {
		t.Fatalf("the tag was not created: %s", resultText(res))
	}
	res = a.call(writer, "delete_tag", map[string]any{"space_id": sp, "id": tag})
	if !res.IsError || !strings.Contains(resultText(res), "tags:delete") {
		t.Fatalf("delete_tag with tags:write = %v %s", res.IsError, resultText(res))
	}

	res = a.call(session("tags:delete"), "delete_tag", map[string]any{"space_id": sp, "id": tag})
	if res.IsError {
		t.Fatalf("delete_tag: %s", resultText(res))
	}
	res = a.call(writer, "list_tags", map[string]any{"space_id": sp})
	if strings.Contains(resultText(res), `"food"`) {
		t.Fatalf("the tag is still there: %s", resultText(res))
	}
}

func TestMCPWriteToolsRecordAndChangeMoney(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	perms, err := auth.PresetPermissions(auth.PresetAdmin, true, auth.MCPResources)
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := a.issueGrant(u, auth.Grant{Permissions: perms, AllSpaces: true}, nil)
	s := a.mcpSession(tok)

	res := a.call(s, "create_account", map[string]any{"space_id": sp, "name": "Wallet", "type": "cash", "currency_code": "EUR", "initial_balance": "100.00"})
	if res.IsError {
		t.Fatalf("create_account: %s", resultText(res))
	}
	acc := idOf(t, res)

	// Amounts travel as exact decimal strings and come back exact.
	res = a.call(s, "create_transaction", map[string]any{"space_id": sp, "type": "expense", "account_id": acc, "amount": "12.34", "description": "lunch", "date": "2026-03-01"})
	if res.IsError || !strings.Contains(resultText(res), "12.34") {
		t.Fatalf("create_transaction: %v %s", res.IsError, resultText(res))
	}
	tx := idOf(t, res)

	res = a.call(s, "update_transaction", map[string]any{"space_id": sp, "id": tx, "amount": "20.00"})
	if res.IsError || !strings.Contains(resultText(res), "lunch") || !strings.Contains(resultText(res), "20") {
		t.Fatalf("update_transaction must change only the amount: %v %s", res.IsError, resultText(res))
	}

	bad := a.call(s, "create_transaction", map[string]any{"space_id": sp, "type": "expense", "account_id": 0, "amount": "1"})
	if !bad.IsError {
		t.Fatalf("a transaction without an account must be refused: %s", resultText(bad))
	}

	if res = a.call(s, "delete_transaction", map[string]any{"space_id": sp, "id": tx}); res.IsError || !strings.Contains(resultText(res), "ok") {
		t.Fatalf("delete_transaction: %v %s", res.IsError, resultText(res))
	}
	if res = a.call(s, "get_transaction", map[string]any{"space_id": sp, "id": tx}); !res.IsError {
		t.Fatalf("the transaction is still there: %s", resultText(res))
	}
}

func TestMCPWriteToolsAreMarkedForClients(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	tok, _ := a.issueGrant(u, auth.Grant{Permissions: []string{"transactions:delete", "accounts:read"}, AllSpaces: true}, nil)
	sess := a.mcpSession(tok)
	list, err := sess.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	hints := map[string]*mcp.ToolAnnotations{}
	for _, tl := range list.Tools {
		hints[tl.Name] = tl.Annotations
	}
	if h := hints["list_accounts"]; h == nil || !h.ReadOnlyHint {
		t.Fatalf("list_accounts must be read-only: %+v", h)
	}
	if h := hints["create_transaction"]; h == nil || h.ReadOnlyHint || (h.DestructiveHint != nil && *h.DestructiveHint) {
		t.Fatalf("create_transaction writes but does not destroy: %+v", h)
	}
	if h := hints["delete_transaction"]; h == nil || h.DestructiveHint == nil || !*h.DestructiveHint {
		t.Fatalf("delete_transaction must be destructive: %+v", h)
	}
}

// idOf reads the id of the item a write tool returned.
func idOf(t *testing.T, res *mcp.CallToolResult) int64 {
	t.Helper()
	var body struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(resultText(res)), &body); err != nil || body.Data.ID == 0 {
		t.Fatalf("no id in %s", resultText(res))
	}
	return body.Data.ID
}

// An assistant is offered only the tools its token can use, so it does not try
// what would fail. Calling one anyway still gets the REST error.
func TestMCPListsOnlyTheToolsTheTokenCanUse(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	names := func(perms ...string) []string {
		tok, _ := a.issueGrant(u, auth.Grant{Permissions: perms, AllSpaces: true}, nil)
		list, err := a.mcpSession(tok).ListTools(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, tl := range list.Tools {
			out = append(out, tl.Name)
		}
		return out
	}
	has := func(list []string, name string) bool { return slices.Contains(list, name) }

	viewer := names("accounts:read", "transactions:read")
	if !has(viewer, "list_spaces") || !has(viewer, "list_accounts") || !has(viewer, "get_transaction") {
		t.Fatalf("a reader is missing its tools: %v", viewer)
	}
	for _, hidden := range []string{"list_tags", "get_report", "create_transaction", "delete_transaction", "create_account"} {
		if has(viewer, hidden) {
			t.Errorf("%s is offered to a token that cannot use it: %v", hidden, viewer)
		}
	}

	editor := names("accounts:read", "transactions:write")
	if !has(editor, "create_transaction") || !has(editor, "update_transaction") || has(editor, "delete_transaction") || has(editor, "create_account") {
		t.Fatalf("write without delete: %v", editor)
	}
	if all := names("transactions:delete"); !has(all, "delete_transaction") {
		t.Fatalf("delete is missing: %v", all)
	}
}

func TestMCPAutomationRules(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mcp@a.com", "secret1", "admin")
	sp := a.space.ID
	perms, err := auth.PresetPermissions(auth.PresetAdmin, true, auth.MCPResources)
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := a.issueGrant(u, auth.Grant{Permissions: perms, AllSpaces: true}, nil)
	s := a.mcpSession(tok)

	res := a.call(s, "create_automation_rule", map[string]any{
		"space_id": sp, "name": "Coffee", "trigger_type": "on_transaction_create",
		"conditions": map[string]any{"match": "all", "conditions": []any{map[string]any{"field": "description", "op": "contains", "value": "coffee"}}},
		"actions":    []any{map[string]any{"type": "set_description", "value": "Coffee"}},
	})
	if res.IsError {
		t.Fatalf("create_automation_rule: %s", resultText(res))
	}
	id := idOf(t, res)

	if res = a.call(s, "list_automation_rules", map[string]any{"space_id": sp}); res.IsError || !strings.Contains(resultText(res), "Coffee") {
		t.Fatalf("list_automation_rules: %v %s", res.IsError, resultText(res))
	}
	if res = a.call(s, "toggle_automation_rule", map[string]any{"space_id": sp, "id": id}); res.IsError {
		t.Fatalf("toggle_automation_rule: %s", resultText(res))
	}
	if res = a.call(s, "get_automation_rule_logs", map[string]any{"space_id": sp, "id": id}); res.IsError {
		t.Fatalf("get_automation_rule_logs: %s", resultText(res))
	}
	if res = a.call(s, "delete_automation_rule", map[string]any{"space_id": sp, "id": id}); res.IsError {
		t.Fatalf("delete_automation_rule: %s", resultText(res))
	}

	// Rules are their own part of a space: tokens for transactions do not see them.
	narrow, _ := a.issueGrant(u, auth.Grant{Permissions: []string{"transactions:write"}, AllSpaces: true}, nil)
	list, _ := a.mcpSession(narrow).ListTools(context.Background(), nil)
	for _, tl := range list.Tools {
		if strings.Contains(tl.Name, "automation") {
			t.Errorf("%s is offered without automation access", tl.Name)
		}
	}
}
