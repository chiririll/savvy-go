package httpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"savvy-go/internal/auth"
	"savvy-go/internal/spaces"
)

// pair is two linked spaces administered by one user: A keeps EUR, B USD.
type pair struct {
	a          *testApp
	owner      *auth.User
	sess       *auth.Issued
	A, B       *spaces.Space
	accA, accB int64
}

func (p *pair) post(spaceID int64, path string, body any) map[string]any {
	p.a.t.Helper()
	res := p.a.do("POST", spacePath(spaceID, path), body, p.sess.Token, p.sess.CSRF)
	out := decodeJSON(p.a.t, res)
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		p.a.t.Fatalf("POST %s %d %v", path, res.StatusCode, out)
	}
	return out
}

func (p *pair) account(spaceID int64, code string) int64 {
	p.a.t.Helper()
	cur := p.post(
		spaceID,
		"/currencies",
		map[string]any{"code": code, "name": code, "symbol": code, "decimals": 2, "rate": 1},
	)
	acc := p.post(
		spaceID,
		"/accounts",
		map[string]any{
			"name":            code + " cash",
			"type":            "cash",
			"currency_id":     cur["data"].(map[string]any)["id"],
			"initial_balance": 0,
		},
	)
	return int64(acc["data"].(map[string]any)["id"].(float64))
}

func newPair(t *testing.T) *pair {
	t.Helper()
	a := newTestApp(t)
	ctx := context.Background()
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	A, err := a.svc.Spaces.Create(ctx, "Personal", owner)
	if err != nil {
		t.Fatal(err)
	}
	B, err := a.svc.Spaces.Create(ctx, "Family", owner)
	if err != nil {
		t.Fatal(err)
	}
	p := &pair{a: a, owner: owner, sess: a.issue(owner, false), A: A, B: B}
	p.accA, p.accB = p.account(A.ID, "EUR"), p.account(B.ID, "USD")
	status(
		t,
		a.do("POST", spacePath(A.ID, "/links"), map[string]any{"target_space_id": B.ID}, p.sess.Token, p.sess.CSRF),
		200,
		"link",
	)
	return p
}

// send transfers from A to B: amount EUR out of A, received USD into B.
func (p *pair) send(amount, received string) string {
	p.a.t.Helper()
	body := p.post(p.A.ID, "/transfers", map[string]any{
		"to_space_id": p.B.ID, "from_account_id": p.accA, "to_account_id": p.accB,
		"from_amount": amount, "to_amount": received, "date": "2026-10-01", "description": "to family",
	})
	return body["data"].(map[string]any)["uuid"].(string)
}

func (p *pair) balance(spaceID, account int64) string {
	p.a.t.Helper()
	res := p.a.do("GET", spacePath(spaceID, "/accounts/"+strconv.FormatInt(account, 10)), nil, p.sess.Token, "")
	body := decodeJSON(p.a.t, res)
	if res.StatusCode != 200 {
		p.a.t.Fatalf("account %d %v", res.StatusCode, body)
	}
	return strconv.FormatFloat(body["data"].(map[string]any)["currentBalance"].(float64), 'f', -1, 64)
}

func (p *pair) transfers(spaceID int64, sess *auth.Issued) map[string]map[string]any {
	p.a.t.Helper()
	res := p.a.do("GET", spacePath(spaceID, "/transfers"), nil, sess.Token, "")
	body := decodeJSON(p.a.t, res)
	if res.StatusCode != 200 {
		p.a.t.Fatalf("transfers %d %v", res.StatusCode, body)
	}
	out := map[string]map[string]any{}
	for _, x := range body["data"].([]any) {
		m := x.(map[string]any)
		out[m["uuid"].(string)] = m
	}
	return out
}

// A transfer moves money out of one space's account and into the
// other's, each in its own currency, and is neither income nor expense.
func TestTransferMovesMoneyBetweenSpaces(t *testing.T) {
	p := newPair(t)
	id := p.send("100", "108.50")
	if b := p.balance(p.A.ID, p.accA); b != "-100" {
		t.Fatalf("A balance %s", b)
	}
	if b := p.balance(p.B.ID, p.accB); b != "108.5" {
		t.Fatalf("B balance %s", b)
	}
	for _, sp := range []int64{p.A.ID, p.B.ID} {
		tr := p.transfers(sp, p.sess)[id]
		if tr == nil || tr["status"] != "ok" || tr["verified"] != true || tr["frozen"] != false {
			t.Fatalf("space %d transfer %v", sp, tr)
		}
	}
	res := p.a.do("GET", spacePath(p.B.ID, "/reports/overview"), nil, p.sess.Token, "")
	if body := decodeJSON(t, res); strings.Contains(strings.ToLower(jsonString(body)), "108.5") {
		t.Fatalf("the transfer shows in B's income or expense: %v", body)
	}
}

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func domainSHA(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// The transaction of a transfer's side changes only through the
// transfer, and its account cannot be deleted.
func TestTransferSidesAreReadOnly(t *testing.T) {
	p := newPair(t)
	p.send("10", "11")
	list := decodeJSON(t, p.a.do("GET", spacePath(p.A.ID, "/transactions"), nil, p.sess.Token, ""))["data"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["type"] != "transfer_out" {
		t.Fatalf("A transactions %v", list)
	}
	for action, allowed := range list[0].(map[string]any)["actions"].(map[string]any) {
		if allowed == true {
			t.Fatalf("the transfer side offers %s", action)
		}
	}
	id := strconv.FormatInt(int64(list[0].(map[string]any)["id"].(float64)), 10)
	status(
		t,
		p.a.do("PATCH", spacePath(p.A.ID, "/transactions/"+id), map[string]any{"amount": 1}, p.sess.Token, p.sess.CSRF),
		422,
		"edit",
	)
	status(t, p.a.do("DELETE", spacePath(p.A.ID, "/transactions/"+id), nil, p.sess.Token, p.sess.CSRF), 422, "delete")
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/transactions/"+id+"/duplicate"), nil, p.sess.Token, p.sess.CSRF),
		422,
		"duplicate",
	)
	status(
		t,
		p.a.do("DELETE", spacePath(p.A.ID, "/accounts/"+strconv.FormatInt(p.accA, 10)), nil, p.sess.Token, p.sess.CSRF),
		422,
		"delete account",
	)
	status(
		t,
		p.a.do(
			"POST",
			spacePath(p.A.ID, "/transactions"),
			map[string]any{"type": "transfer_out", "account_id": p.accA, "amount": 1, "date": "2026-10-01"},
			p.sess.Token,
			p.sess.CSRF,
		),
		422,
		"create a side by hand",
	)
}

// A member of only one space sees the other space's name and amount,
// never its account.
func TestOtherSideAccountIsPrivate(t *testing.T) {
	p := newPair(t)
	id := p.send("5", "5")
	bob := p.a.createUser("bob@test.com", "secret1", auth.RoleUser)
	_ = p.a.svc.Spaces.SetMember(context.Background(), p.B.ID, bob, spaces.RoleViewer)
	bs := p.a.issue(bob, false)
	seen := p.transfers(p.B.ID, bs)[id]
	from := seen["from"].(map[string]any)
	if _, ok := from["accountId"]; ok {
		t.Fatalf("bob sees A's account: %v", seen)
	}
	if seen["otherSpace"].(map[string]any)["name"] != "Personal" || from["amount"] != "5" {
		t.Fatalf("bob's view %v", seen)
	}
	if _, ok := p.transfers(p.B.ID, p.sess)[id]["from"].(map[string]any)["accountId"]; !ok {
		t.Fatal("a member of both spaces should see both accounts")
	}
	status(t, p.a.do("POST", spacePath(p.B.ID, "/transfers"), map[string]any{}, bs.Token, bs.CSRF), 403, "viewer sends")
}

// Only who administers both spaces links them; a space the caller is
// not in is a 404; an admin of either side unlinks.
func TestLinkRights(t *testing.T) {
	p := newPair(t)
	ctx := context.Background()
	other := p.a.createUser("other@test.com", "secret1", auth.RoleUser)
	C, _ := p.a.svc.Spaces.Create(ctx, "Theirs", other)
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": C.ID}, p.sess.Token, p.sess.CSRF),
		404,
		"not a member of the target",
	)
	_ = p.a.svc.Spaces.SetMember(ctx, C.ID, p.owner, spaces.RoleEditor)
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": C.ID}, p.sess.Token, p.sess.CSRF),
		403,
		"editor of the target",
	)
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": p.A.ID}, p.sess.Token, p.sess.CSRF),
		422,
		"link to itself",
	)

	links := decodeJSON(t, p.a.do("GET", spacePath(p.B.ID, "/links"), nil, p.sess.Token, ""))["data"].([]any)
	if len(links) != 1 || links[0].(map[string]any)["name"] != "Personal" {
		t.Fatalf("links of B %v", links)
	}
	status(
		t,
		p.a.do("DELETE", spacePath(p.B.ID, "/links/"+strconv.FormatInt(p.A.ID, 10)), nil, p.sess.Token, p.sess.CSRF),
		204,
		"unlink from B",
	)
}

// Unlinking freezes the transfers in both spaces (they still count);
// linking again unfreezes them and merges what changed.
func TestUnlinkFreezesRelinkSyncs(t *testing.T) {
	p := newPair(t)
	id := p.send("20", "20")
	status(
		t,
		p.a.do("DELETE", spacePath(p.A.ID, "/links/"+strconv.FormatInt(p.B.ID, 10)), nil, p.sess.Token, p.sess.CSRF),
		204,
		"unlink",
	)
	for _, sp := range []int64{p.A.ID, p.B.ID} {
		if tr := p.transfers(sp, p.sess)[id]; tr["frozen"] != true {
			t.Fatalf("space %d not frozen: %v", sp, tr)
		}
	}
	if b := p.balance(p.A.ID, p.accA); b != "-20" {
		t.Fatalf("frozen transfer left the balance: %s", b)
	}
	status(
		t,
		p.a.do("PATCH", spacePath(p.A.ID, "/transfers/"+id), map[string]any{"from_amount": "1"}, p.sess.Token, p.sess.CSRF),
		422,
		"edit frozen",
	)
	status(
		t,
		p.a.do(
			"POST",
			spacePath(p.A.ID, "/transfers"),
			map[string]any{
				"to_space_id":     p.B.ID,
				"from_account_id": p.accA,
				"to_account_id":   p.accB,
				"from_amount":     "1",
				"to_amount":       "1",
				"date":            "2026-10-01",
			},
			p.sess.Token,
			p.sess.CSRF,
		),
		422,
		"send unlinked",
	)

	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": p.B.ID}, p.sess.Token, p.sess.CSRF),
		200,
		"relink",
	)
	if tr := p.transfers(p.B.ID, p.sess)[id]; tr["frozen"] != false {
		t.Fatalf("not unfrozen %v", tr)
	}
	status(
		t,
		p.a.do(
			"PATCH",
			spacePath(p.A.ID, "/transfers/"+id),
			map[string]any{"from_amount": "25", "to_amount": "26"},
			p.sess.Token,
			p.sess.CSRF,
		),
		200,
		"edit after relink",
	)
	if b := p.balance(p.B.ID, p.accB); b != "26" {
		t.Fatalf("B balance after edit %s", b)
	}
}

// When one space cannot take the write (quota), neither changes.
func TestTransferIntoFullSpaceWritesNothing(t *testing.T) {
	p := newPair(t)
	ctx := context.Background()
	if err := p.a.svc.Spaces.SetQuota(ctx, p.owner, p.B.ID, 1); err != nil {
		t.Fatal(err)
	}
	var code int
	for i := 0; i < 200; i++ {
		res := p.a.do(
			"POST",
			spacePath(p.B.ID, "/tags"),
			map[string]any{"name": strings.Repeat("x", 900) + strconv.Itoa(i)},
			p.sess.Token,
			p.sess.CSRF,
		)
		code = res.StatusCode
		res.Body.Close()
		if code != 201 {
			break
		}
	}
	if code != 422 {
		t.Fatalf("B never filled up: %d", code)
	}
	res := p.a.do("POST", spacePath(p.A.ID, "/transfers"), map[string]any{
		"to_space_id": p.B.ID, "from_account_id": p.accA, "to_account_id": p.accB, "from_amount": "1", "to_amount": "1",
		"date":        "2026-10-01",
		"description": strings.Repeat("d", 4000),
	}, p.sess.Token, p.sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 422 {
		t.Fatalf("transfer into a full space %d %v", res.StatusCode, body)
	}
	if n := len(p.transfers(p.A.ID, p.sess)); n != 0 {
		t.Fatalf("A kept %d transfers", n)
	}
}

// Transfers do not run automation rules.
func TestTransfersSkipAutomation(t *testing.T) {
	p := newPair(t)
	p.post(p.B.ID, "/automation-rules", map[string]any{
		"name": "Every incoming", "trigger_type": "on_transaction_create", "priority": 10,
		"conditions": map[string]any{
			"match":      "all",
			"conditions": []map[string]any{{"field": "amount", "op": "gte", "value": 0}},
		},
		"actions":   []map[string]any{{"type": "set_description", "value": "touched"}},
		"is_active": true,
	})
	p.send("3", "3")
	list := decodeJSON(t, p.a.do("GET", spacePath(p.B.ID, "/transactions"), nil, p.sess.Token, ""))["data"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["description"] != "to family" {
		t.Fatalf("automation touched the transfer: %v", list)
	}
}

// --- restore and review --------------------------------------------------------

// After A is restored, A's ids count up again, so a new transaction can
// take the id the transfer's side had. The side is tied to the transfer by
// uuid, never by id: the new transaction stays an ordinary one, and the
// transfer comes back with a side of its own.
func TestReusedIDIsNotTheTransfer(t *testing.T) {
	p := newPair(t)
	backup := p.backupA()
	id := p.send("5", "5")
	sideID := decodeJSON(t,
		p.a.do("GET", spacePath(p.A.ID, "/transactions"), nil, p.sess.Token, ""))["data"].([]any)[0].(map[string]any)["id"]
	// Unlinked, the restore has nothing to review and A forgets the transfer.
	status(
		t,
		p.a.do("DELETE", spacePath(p.A.ID, "/links/"+strconv.FormatInt(p.B.ID, 10)), nil, p.sess.Token, p.sess.CSRF),
		204,
		"unlink",
	)
	p.restoreA(backup)
	created := p.post(
		p.A.ID,
		"/transactions",
		map[string]any{"type": "expense", "account_id": p.accA, "amount": 1, "date": "2026-10-01"},
	)
	newID := created["data"].(map[string]any)["id"]
	if newID != sideID {
		t.Fatalf("the scenario needs the id reused: side %v, new %v", sideID, newID)
	}
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": p.B.ID}, p.sess.Token, p.sess.CSRF),
		200,
		"relink",
	)

	path := spacePath(p.A.ID, "/transactions/"+strconv.FormatFloat(newID.(float64), 'f', -1, 64))
	status(
		t,
		p.a.do(
			"PATCH",
			path,
			map[string]any{"type": "expense", "account_id": p.accA, "amount": 2, "date": "2026-10-01"},
			p.sess.Token,
			p.sess.CSRF,
		),
		200,
		"the new transaction is ordinary",
	)
	if tr := p.transfers(p.A.ID, p.sess)[id]; tr == nil || tr["status"] != "ok" {
		t.Fatalf("the transfer did not come back: %v", tr)
	}
	if b := p.balance(p.A.ID, p.accA); b != "-7" {
		t.Fatalf("A balance %s, want the expense (2) and the transfer (5)", b)
	}
}

// A write that breaks a database constraint answers 422 without the schema.
func TestConstraintErrorHidesSchema(t *testing.T) {
	p := newPair(t)
	res := p.a.do(
		"POST",
		spacePath(p.A.ID, "/transactions"),
		map[string]any{"type": "bogus", "account_id": p.accA, "amount": 1, "date": "2026-10-01"},
		p.sess.Token,
		p.sess.CSRF,
	)
	body := toString(decodeJSON(t, res))
	if res.StatusCode != 422 {
		t.Fatalf("status %d, want 422: %s", res.StatusCode, body)
	}
	for _, leak := range []string{"CHECK", "constraint", "IN (", "SQL"} {
		if strings.Contains(body, leak) {
			t.Errorf("response leaks %q: %s", leak, body)
		}
	}
}
