package httpserver

import (
	"context"
	"net/http"
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

func TestMCPToolsAreAllReadOnly(t *testing.T) {
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
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %s is not marked read-only", tool.Name)
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
