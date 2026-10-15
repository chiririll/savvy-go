package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"savvy-go/internal/auth"
)

// Every data route must belong to a resource, or tokens are locked out of it
// (and a forgotten one would have silently opened to every token). A route
// added to dataRoutes without a spaceSegments entry fails here.
func TestEveryDataRouteMapsToAResource(t *testing.T) {
	a := newTestApp(t)
	for root := range dataRoots(a.s) {
		res, ok := spaceResource("/api/spaces/1/" + root)
		if !ok {
			t.Errorf("data route /%s has no API token resource: add it to spaceSegments", root)
		} else if !slices.Contains(auth.APIResources, res) {
			t.Errorf("/%s maps to %q, which is not an API resource", root, res)
		}
	}
	for _, path := range []string{"/api/spaces/1", "/api/spaces/1/", "/api/spaces/1/members", "/api/spaces/1/transfers/abc/accept"} {
		if _, ok := spaceResource(path); !ok {
			t.Errorf("%s has no resource", path)
		}
	}
	for _, path := range []string{"/api/spaces/1/audit", "/api/spaces/1/backups", "/api/spaces/1/invitations", "/api/spaces/1/nope"} {
		if res, ok := spaceResource(path); ok {
			t.Errorf("%s maps to %q: tokens must not reach it", path, res)
		}
	}
}

func TestTokenPermissionsAreCheckedPerResource(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("perm@a.com", "secret1", roleEditor)

	readAccounts, _ := a.issueGrant(u, auth.Grant{Permissions: []string{"accounts:read"}, AllSpaces: true}, nil)
	status(t, a.bearer("GET", "/api/accounts", nil, readAccounts), 200, "granted read")
	status(t, a.bearer("GET", "/api/transactions", nil, readAccounts), 403, "other resource")
	status(t, a.bearer("GET", "/api/reports/overview", nil, readAccounts), 403, "reports are their own resource")
	res := a.bearer("POST", "/api/accounts", map[string]any{"name": "X"}, readAccounts)
	if res.StatusCode != 403 {
		t.Fatalf("write with a read grant = %d", res.StatusCode)
	}
	if msg := decodeJSON(t, res)["message"]; msg != "This API token has no accounts:write permission." {
		t.Fatalf("message = %v", msg)
	}

	writeTags, _ := a.issueGrant(u, auth.Grant{Permissions: []string{"tags:write"}, AllSpaces: true}, nil)
	status(t, a.bearer("POST", "/api/tags", map[string]any{"name": "t"}, writeTags), 201, "granted write")
	status(t, a.bearer("GET", "/api/tags", nil, writeTags), 200, "write includes read")
	status(t, a.bearer("GET", "/api/accounts", nil, writeTags), 403, "write on one resource opens nothing else")

	// No grants at all: the token can list its spaces and nothing inside them.
	none, _ := a.issueGrant(u, auth.Grant{AllSpaces: true}, nil)
	status(t, a.bearer("GET", "/api/spaces", nil, none), 200, "list spaces")
	status(t, a.bearer("GET", "/api/accounts", nil, none), 403, "no permissions")
	status(t, a.bearer("GET", fmt.Sprintf("/api/spaces/%d", a.space.ID), nil, none), 403, "space info needs space:read")

	// A grant is still held to the owner's role.
	viewer := a.createUser("viewer@a.com", "secret1", roleViewer)
	viewerWrite, _ := a.issueGrant(viewer, auth.Grant{Permissions: []string{"accounts:write"}, AllSpaces: true}, nil)
	status(t, a.bearer("POST", "/api/accounts", map[string]any{"name": "X"}, viewerWrite), 403, "viewer role wins over the grant")

	// Spaces' own administration stays out of reach whatever the grant.
	all, _ := a.issueGrant(u, auth.Grant{Permissions: allPerms(auth.PermWrite), AllSpaces: true}, nil)
	for _, p := range []string{"/audit", "/backups", "/invitations"} {
		status(t, a.bearer("GET", fmt.Sprintf("/api/spaces/%d%s", a.space.ID, p), nil, all), 403, "tokens cannot reach "+p)
	}
}

func TestTokenReachesOnlyItsSpaces(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("two@a.com", "secret1", roleEditor)
	other, err := a.s.spaces.Create(context.Background(), "Other", u)
	if err != nil {
		t.Fatal(err)
	}
	inShared := fmt.Sprintf("/api/spaces/%d/accounts", a.space.ID)
	inOther := fmt.Sprintf("/api/spaces/%d/accounts", other.ID)

	one, _ := a.issueGrant(u, auth.Grant{Permissions: []string{"accounts:write"}, SpaceIDs: []int64{other.ID}}, nil)
	status(t, a.bearer("GET", inOther, nil, one), 200, "its space")
	status(t, a.bearer("POST", inOther, map[string]any{"name": "X", "type": "bank", "currency_code": "GBP", "initial_balance": 0}, one), 201, "write in its space")
	status(t, a.bearer("GET", inShared, nil, one), 404, "a space of the owner it was not given looks like it does not exist")
	status(t, a.bearer("POST", inShared, map[string]any{"name": "X"}, one), 404, "no writing there either")

	list := decodeJSON(t, a.bearer("GET", "/api/spaces", nil, one))["data"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["id"] != float64(other.ID) {
		t.Fatalf("the token lists %v, want only its space", list)
	}

	everywhere, _ := a.issueGrant(u, auth.Grant{Permissions: []string{"accounts:read"}, AllSpaces: true}, nil)
	status(t, a.bearer("GET", inShared, nil, everywhere), 200, "all spaces: shared")
	status(t, a.bearer("GET", inOther, nil, everywhere), 200, "all spaces: other")
	if list := decodeJSON(t, a.bearer("GET", "/api/spaces", nil, everywhere))["data"].([]any); len(list) != 2 {
		t.Fatalf("all-spaces token lists %v", list)
	}

	// Losing the space ends the token's reach, it does not widen it.
	if _, err := a.db.Exec(`DELETE FROM api_token_spaces`); err != nil {
		t.Fatal(err)
	}
	status(t, a.bearer("GET", inOther, nil, one), 404, "a token left with no spaces reaches none")
}

func TestCreateTokenKeepsTheChosenSpaces(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("mk@a.com", "secret1", roleEditor)
	sess := a.issue(u, false)

	res := a.do("POST", "/api/auth/api-tokens", map[string]any{
		"name": "ci", "permissions": []string{"transactions:write", "transactions:read", "accounts:read"}, "space_ids": []int64{a.space.ID},
	}, sess.Token, sess.CSRF)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create %d", res.StatusCode)
	}
	data := decodeJSON(t, res)["data"].(map[string]any)
	if fmt.Sprint(data["permissions"]) != "[accounts:read transactions:write]" || data["all_spaces"] != false || fmt.Sprint(data["space_ids"]) != fmt.Sprint([]any{float64(a.space.ID)}) {
		t.Fatalf("created = %v", data)
	}
	items := decodeJSON(t, a.do("GET", "/api/auth/api-tokens", nil, sess.Token, ""))["data"].([]any)
	if got := items[0].(map[string]any); fmt.Sprint(got["space_ids"]) != fmt.Sprint(data["space_ids"]) || got["all_spaces"] != false {
		t.Fatalf("listed = %v", got)
	}
}

// The consent screen decides which spaces an assistant reads; the token it gets
// reads what the MCP tools show and nothing more, and keeps its spaces through
// every refresh.
func TestOAuthConsentChoosesTheSpaces(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("oauth@a.com", "secret1", roleEditor)
	other, err := a.s.spaces.Create(context.Background(), "Other", u)
	if err != nil {
		t.Fatal(err)
	}
	sess := a.issue(u, false)
	id := a.registerClient()

	info := decodeJSON(t, a.do("GET", "/api/oauth/authorize?client_id="+id+"&redirect_uri="+url.QueryEscape(oauthRedirect), nil, sess.Token, sess.CSRF))["data"].(map[string]any)
	if spaces, _ := info["spaces"].([]any); len(spaces) != 2 {
		t.Fatalf("consent lists spaces %v", info["spaces"])
	}
	if res, _ := info["resources"].([]any); len(res) != len(auth.MCPResources) {
		t.Fatalf("consent lists resources %v", info["resources"])
	}

	decide := func(body map[string]any) *http.Response {
		body["client_id"], body["redirect_uri"], body["state"] = id, oauthRedirect, "st-1"
		body["code_challenge"], body["code_challenge_method"], body["approve"] = oauthChallenge(), "S256", true
		return a.do("POST", "/api/oauth/authorize", body, sess.Token, sess.CSRF)
	}
	status(t, decide(map[string]any{}), 422, "no choice")
	status(t, decide(map[string]any{"all_spaces": true, "space_ids": []int64{other.ID}}), 422, "both")
	status(t, decide(map[string]any{"space_ids": []int64{424242}}), 422, "a space the user is not in")

	res := decide(map[string]any{"space_ids": []int64{other.ID}})
	if res.StatusCode != 200 {
		t.Fatalf("approve = %d", res.StatusCode)
	}
	code := codeFrom(t, decodeJSON(t, res)["data"].(map[string]any)["redirect_url"].(string))
	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {id}, "code": {code}, "code_verifier": {oauthVerifier}, "redirect_uri": {oauthRedirect}}
	st, tok := a.token(form)
	if st != 200 {
		t.Fatalf("token = %d %v", st, tok)
	}

	check := func(access string) {
		t.Helper()
		status(t, a.bearer("GET", fmt.Sprintf("/api/spaces/%d/accounts", other.ID), nil, access), 200, "chosen space")
		status(t, a.bearer("GET", fmt.Sprintf("/api/spaces/%d/accounts", a.space.ID), nil, access), 404, "space that was not chosen")
		status(t, a.bearer("GET", fmt.Sprintf("/api/spaces/%d/transfers", other.ID), nil, access), 403, "transfers are not for assistants")
		status(t, a.bearer("GET", fmt.Sprintf("/api/spaces/%d/members", other.ID), nil, access), 403, "members are not for assistants")
		status(t, a.bearer("POST", fmt.Sprintf("/api/spaces/%d/accounts", other.ID), map[string]any{"name": "X"}, access), 403, "read only")
	}
	check(tok["access_token"].(string))

	st, next := a.token(url.Values{"grant_type": {"refresh_token"}, "client_id": {id}, "refresh_token": {tok["refresh_token"].(string)}})
	if st != 200 {
		t.Fatalf("refresh = %d %v", st, next)
	}
	check(next["access_token"].(string))

	// The user can see what the connection holds in their token list.
	items := decodeJSON(t, a.do("GET", "/api/auth/api-tokens", nil, sess.Token, ""))["data"].([]any)
	if len(items) != 1 || strings.Contains(fmt.Sprint(items[0].(map[string]any)["permissions"]), "write") || items[0].(map[string]any)["all_spaces"] != false {
		t.Fatalf("connection token = %v", items)
	}
}
