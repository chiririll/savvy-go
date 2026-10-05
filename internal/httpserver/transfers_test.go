package httpserver

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/domain"
)

// pair is two linked spaces administered by one user: A keeps EUR, B USD.
type pair struct {
	a          *testApp
	owner      *auth.User
	sess       *auth.Issued
	A, B       *domain.Space
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
	cur := p.post(spaceID, "/currencies", map[string]any{"code": code, "name": code, "symbol": code, "decimals": 2, "rate": 1})
	acc := p.post(spaceID, "/accounts", map[string]any{"name": code + " cash", "type": "cash", "currency_id": cur["data"].(map[string]any)["id"], "initial_balance": 0})
	return int64(acc["data"].(map[string]any)["id"].(float64))
}

func newPair(t *testing.T) *pair {
	t.Helper()
	a := newTestApp(t)
	ctx := context.Background()
	owner := a.createUser("owner@test.com", "secret1", auth.RoleUser)
	A, err := a.s.spaces.Create(ctx, "Personal", owner)
	if err != nil {
		t.Fatal(err)
	}
	B, err := a.s.spaces.Create(ctx, "Family", owner)
	if err != nil {
		t.Fatal(err)
	}
	p := &pair{a: a, owner: owner, sess: a.issue(owner, false), A: A, B: B}
	p.accA, p.accB = p.account(A.ID, "EUR"), p.account(B.ID, "USD")
	status(t, a.do("POST", spacePath(A.ID, "/links"), map[string]any{"target_space_id": B.ID}, p.sess.Token, p.sess.CSRF), 200, "link")
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

// P32: a transfer moves money out of one space's account and into the
// other's, each in its own currency, and is neither income nor expense.
func TestP32TransferMovesMoneyBetweenSpaces(t *testing.T) {
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

// P31: the transaction of a transfer's side changes only through the
// transfer, and its account cannot be deleted.
func TestP31TransferRowIsReadOnly(t *testing.T) {
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
	status(t, p.a.do("PATCH", spacePath(p.A.ID, "/transactions/"+id), map[string]any{"amount": 1}, p.sess.Token, p.sess.CSRF), 422, "edit")
	status(t, p.a.do("DELETE", spacePath(p.A.ID, "/transactions/"+id), nil, p.sess.Token, p.sess.CSRF), 422, "delete")
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transactions/"+id+"/duplicate"), nil, p.sess.Token, p.sess.CSRF), 422, "duplicate")
	status(t, p.a.do("DELETE", spacePath(p.A.ID, "/accounts/"+strconv.FormatInt(p.accA, 10)), nil, p.sess.Token, p.sess.CSRF), 422, "delete account")
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transactions"), map[string]any{"type": "transfer_out", "account_id": p.accA, "amount": 1, "date": "2026-10-01"}, p.sess.Token, p.sess.CSRF), 422, "create a side by hand")
}

// P30: a member of only one space sees the other space's name and amount,
// never its account.
func TestP30OtherSideAccountIsPrivate(t *testing.T) {
	p := newPair(t)
	id := p.send("5", "5")
	bob := p.a.createUser("bob@test.com", "secret1", auth.RoleUser)
	_ = p.a.s.spaces.SetMember(context.Background(), p.B.ID, bob, domain.SpaceViewer)
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

// P52: only who administers both spaces links them; a space the caller is
// not in is a 404; an admin of either side unlinks.
func TestP52LinkRights(t *testing.T) {
	p := newPair(t)
	ctx := context.Background()
	other := p.a.createUser("other@test.com", "secret1", auth.RoleUser)
	C, _ := p.a.s.spaces.Create(ctx, "Theirs", other)
	status(t, p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": C.ID}, p.sess.Token, p.sess.CSRF), 404, "not a member of the target")
	_ = p.a.s.spaces.SetMember(ctx, C.ID, p.owner, domain.SpaceEditor)
	status(t, p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": C.ID}, p.sess.Token, p.sess.CSRF), 403, "editor of the target")
	status(t, p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": p.A.ID}, p.sess.Token, p.sess.CSRF), 422, "link to itself")

	links := decodeJSON(t, p.a.do("GET", spacePath(p.B.ID, "/links"), nil, p.sess.Token, ""))["data"].([]any)
	if len(links) != 1 || links[0].(map[string]any)["name"] != "Personal" {
		t.Fatalf("links of B %v", links)
	}
	status(t, p.a.do("DELETE", spacePath(p.B.ID, "/links/"+strconv.FormatInt(p.A.ID, 10)), nil, p.sess.Token, p.sess.CSRF), 204, "unlink from B")
}

// P28: unlinking freezes the transfers in both spaces (they still count);
// linking again unfreezes them and merges what changed.
func TestP28UnlinkFreezesRelinkSyncs(t *testing.T) {
	p := newPair(t)
	id := p.send("20", "20")
	status(t, p.a.do("DELETE", spacePath(p.A.ID, "/links/"+strconv.FormatInt(p.B.ID, 10)), nil, p.sess.Token, p.sess.CSRF), 204, "unlink")
	for _, sp := range []int64{p.A.ID, p.B.ID} {
		if tr := p.transfers(sp, p.sess)[id]; tr["frozen"] != true {
			t.Fatalf("space %d not frozen: %v", sp, tr)
		}
	}
	if b := p.balance(p.A.ID, p.accA); b != "-20" {
		t.Fatalf("frozen transfer left the balance: %s", b)
	}
	status(t, p.a.do("PATCH", spacePath(p.A.ID, "/transfers/"+id), map[string]any{"from_amount": "1"}, p.sess.Token, p.sess.CSRF), 422, "edit frozen")
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transfers"), map[string]any{"to_space_id": p.B.ID, "from_account_id": p.accA, "to_account_id": p.accB, "from_amount": "1", "to_amount": "1", "date": "2026-10-01"}, p.sess.Token, p.sess.CSRF), 422, "send unlinked")

	status(t, p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": p.B.ID}, p.sess.Token, p.sess.CSRF), 200, "relink")
	if tr := p.transfers(p.B.ID, p.sess)[id]; tr["frozen"] != false {
		t.Fatalf("not unfrozen %v", tr)
	}
	status(t, p.a.do("PATCH", spacePath(p.A.ID, "/transfers/"+id), map[string]any{"from_amount": "25", "to_amount": "26"}, p.sess.Token, p.sess.CSRF), 200, "edit after relink")
	if b := p.balance(p.B.ID, p.accB); b != "26" {
		t.Fatalf("B balance after edit %s", b)
	}
}

// P29: when one space cannot take the write (quota), neither changes.
func TestP29TransferBlockedByQuotaWritesNothing(t *testing.T) {
	p := newPair(t)
	ctx := context.Background()
	if err := p.a.s.spaces.SetQuota(ctx, p.owner, p.B.ID, 1); err != nil {
		t.Fatal(err)
	}
	var code int
	for i := 0; i < 200; i++ {
		res := p.a.do("POST", spacePath(p.B.ID, "/tags"), map[string]any{"name": strings.Repeat("x", 900) + strconv.Itoa(i)}, p.sess.Token, p.sess.CSRF)
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
		"to_space_id": p.B.ID, "from_account_id": p.accA, "to_account_id": p.accB, "from_amount": "1", "to_amount": "1", "date": "2026-10-01",
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

// P40: transfers do not run automation rules.
func TestP40TransfersSkipAutomation(t *testing.T) {
	p := newPair(t)
	p.post(p.B.ID, "/automation-rules", map[string]any{
		"name": "Every incoming", "trigger_type": "on_transaction_create", "priority": 10,
		"conditions": map[string]any{"match": "all", "conditions": []map[string]any{{"field": "amount", "op": "gte", "value": 0}}},
		"actions":    []map[string]any{{"type": "set_description", "value": "touched"}},
		"is_active":  true,
	})
	p.send("3", "3")
	list := decodeJSON(t, p.a.do("GET", spacePath(p.B.ID, "/transactions"), nil, p.sess.Token, ""))["data"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["description"] != "to family" {
		t.Fatalf("automation touched the transfer: %v", list)
	}
}

// --- restore and review --------------------------------------------------------

func (p *pair) backupA() string {
	p.a.t.Helper()
	b := createdBackup(p.a.t, p.a.do("POST", spacePath(p.A.ID, "/backups"), map[string]any{}, p.sess.Token, p.sess.CSRF))
	return b["filename"].(string)
}

func (p *pair) restoreA(name string) {
	p.a.t.Helper()
	res := p.a.do("POST", spacePath(p.A.ID, "/backups/"+name+"/restore"), nil, p.sess.Token, p.sess.CSRF)
	if res.StatusCode != 200 {
		p.a.t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(p.a.t, res))
	}
	res.Body.Close()
}

// P24: after A is restored, every difference with B is pending in A and B
// does not change until someone decides; pending changes stay out of A's
// balance, which matches the backup.
func TestP24RestoreReviewsTransfers(t *testing.T) {
	p := newPair(t)
	kept := p.send("1", "1")
	changed := p.send("2", "2")
	deleted := p.send("4", "4")
	backup := p.backupA()
	balanceAtBackup := p.balance(p.A.ID, p.accA)

	created := p.send("8", "8")
	status(t, p.a.do("PATCH", spacePath(p.B.ID, "/transfers/"+changed), map[string]any{"from_amount": "20", "to_amount": "20"}, p.sess.Token, p.sess.CSRF), 200, "change")
	status(t, p.a.do("DELETE", spacePath(p.B.ID, "/transfers/"+deleted), nil, p.sess.Token, p.sess.CSRF), 204, "delete")
	bBefore := p.transfers(p.B.ID, p.sess)
	bBalance := p.balance(p.B.ID, p.accB)

	p.restoreA(backup)

	inA := p.transfers(p.A.ID, p.sess)
	for id, review := range map[string]string{created: "created_remote", changed: "changed_remote", deleted: "deleted_remote"} {
		if tr := inA[id]; tr == nil || tr["status"] != "pending" || tr["review"] != review {
			t.Fatalf("%s in A: %v", review, tr)
		}
	}
	if inA[kept]["status"] != "ok" {
		t.Fatalf("unchanged transfer %v", inA[kept])
	}
	if b := p.balance(p.A.ID, p.accA); b != balanceAtBackup {
		t.Fatalf("A balance %s, the backup had %s", b, balanceAtBackup)
	}
	if b := p.balance(p.B.ID, p.accB); b != bBalance || len(p.transfers(p.B.ID, p.sess)) != len(bBefore) {
		t.Fatal("restoring A changed B")
	}

	// A member without rights in B can only take B's version.
	viewer := p.a.createUser("v@test.com", "secret1", auth.RoleUser)
	_ = p.a.s.spaces.SetMember(context.Background(), p.A.ID, viewer, domain.SpaceEditor)
	vs := p.a.issue(viewer, false)
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transfers/"+changed+"/reject"), nil, vs.Token, vs.CSRF), 403, "reject without rights in B")

	status(t, p.a.do("POST", spacePath(p.A.ID, "/transfers/"+created+"/accept"), nil, p.sess.Token, p.sess.CSRF), 200, "accept created")
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transfers/"+deleted+"/accept"), nil, p.sess.Token, p.sess.CSRF), 204, "accept deleted")
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transfers/"+changed+"/reject"), nil, p.sess.Token, p.sess.CSRF), 200, "reject changed")

	inA = p.transfers(p.A.ID, p.sess)
	if inA[created]["status"] != "ok" || inA[deleted] != nil || inA[changed]["status"] != "ok" {
		t.Fatalf("after decisions %v", inA)
	}
	if amt := p.transfers(p.B.ID, p.sess)[changed]["to"].(map[string]any)["amount"]; amt != "2" {
		t.Fatalf("rejecting did not push A's version to B: %v", amt)
	}
	if b := p.balance(p.A.ID, p.accA); b != "-11" {
		t.Fatalf("A balance after decisions %s", b)
	}
}

// P24: a crash before the review merge finishes reviews again at startup
// instead of merging automatically.
func TestP24ReviewMarkSurvivesCrash(t *testing.T) {
	p := newPair(t)
	backup := p.backupA()
	created := p.send("8", "8")
	ctx := context.Background()
	space, _ := p.a.s.spaces.Get(ctx, p.A.ID)

	// Restore without the review merge, as if the process died right after
	// the swap: the mark is in the file.
	b := p.a.s.backups
	b.Transfers = nil
	if err := b.RestoreSpace(ctx, *space, backup, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.a.s.transfers.SyncAll(ctx); err != nil {
		t.Fatal(err)
	}
	if tr := p.transfers(p.A.ID, p.sess)[created]; tr == nil || tr["status"] != "pending" {
		t.Fatalf("startup merged automatically: %v", tr)
	}
}

// P25: a transfer whose account is not in the restored space needs attention
// until another account is chosen.
func TestP25MissingAccountNeedsAttention(t *testing.T) {
	p := newPair(t)
	backup := p.backupA()
	later := p.account(p.A.ID, "GBP")
	body := p.post(p.A.ID, "/transfers", map[string]any{
		"to_space_id": p.B.ID, "from_account_id": later, "to_account_id": p.accB, "from_amount": "7", "to_amount": "7", "date": "2026-10-01",
	})
	id := body["data"].(map[string]any)["uuid"].(string)
	p.restoreA(backup)
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transfers/"+id+"/accept"), nil, p.sess.Token, p.sess.CSRF), 200, "accept")
	if tr := p.transfers(p.A.ID, p.sess)[id]; tr["status"] != "needs_attention" {
		t.Fatalf("status %v", tr)
	}
	status(t, p.a.do("PATCH", spacePath(p.A.ID, "/transfers/"+id), map[string]any{"from_account_id": p.accA}, p.sess.Token, p.sess.CSRF), 200, "choose account")
	if tr := p.transfers(p.A.ID, p.sess)[id]; tr["status"] != "ok" {
		t.Fatalf("status after fix %v", tr)
	}
}

// forgeBackup rewrites the database inside a space backup with sql run on
// it, and returns the new archive (its manifest no longer matches the
// signature, so it is unsigned).
func forgeBackup(t *testing.T, path string, statements ...string) []byte {
	t.Helper()
	dir := t.TempDir()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "space.sqlite")
	for _, f := range zr.File {
		if f.Name == "space.sqlite" {
			r, _ := f.Open()
			raw, _ := io.ReadAll(r)
			r.Close()
			_ = os.WriteFile(dbPath, raw, 0o644)
		}
	}
	zr.Close()
	d, err := db.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range statements {
		if _, err := d.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	_, _ = d.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	_ = d.Close()
	raw, _ := os.ReadFile(dbPath)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("space.sqlite")
	_, _ = w.Write(raw)
	sum, _ := domainSHA(dbPath)
	mw, _ := zw.Create("manifest.json")
	_, _ = mw.Write([]byte(`{"format":1,"kind":"space","store":"sqlite","files":{"space.sqlite":"` + sum + `"},"space_uuid":"x"}`))
	_ = zw.Close()
	return buf.Bytes()
}

// P37, P38, P43, P51: a doctored backup cannot reach the other space: a
// made-up transfer and a raised version stay pending in A with no way to
// push them, and a genuine record of other spaces is dropped.
func TestP37ForgedBackupCannotReachOtherSpace(t *testing.T) {
	p := newPair(t)
	real := p.send("5", "5")
	backup := p.backupA()
	uuidA, uuidB := p.A.UUID, p.B.UUID
	forged := forgeBackup(t, p.a.s.backups.SpacePath(p.A.ID, backup),
		`UPDATE space_transfers SET version = 99, from_amount = 1 WHERE uuid = '`+real+`'`,
		`INSERT INTO space_transfers (uuid, from_space_uuid, from_account_id, from_amount, from_currency, from_decimals,
			to_space_uuid, to_account_id, to_amount, to_currency, to_decimals, date, version, signer_kid, signature)
			VALUES ('forged', '`+uuidA+`', 1, 100000, 'EUR', 2, '`+uuidB+`', 1, 100000, 'USD', 2, '2026-10-01', 1, 'nobody', 'AAAA')`,
		`INSERT INTO space_transfers (uuid, from_space_uuid, from_account_id, from_amount, from_currency, from_decimals,
			to_space_uuid, to_account_id, to_amount, to_currency, to_decimals, date, version, signer_kid, signature)
			VALUES ('foreign', 'someone', 1, 1, 'EUR', 2, '`+uuidB+`', 1, 1, 'USD', 2, '2026-10-01', 1, 'nobody', 'AAAA')`,
	)
	up := createdBackup(t, p.a.upload(spacePath(p.A.ID, "/backups/upload"), "forged.zip", forged, nil, p.sess))
	// The manifest names another uuid: restore checks the database, not it.
	bBefore := p.balance(p.B.ID, p.accB)
	res := p.a.do("POST", spacePath(p.A.ID, "/backups/"+up["filename"].(string)+"/restore"), nil, p.sess.Token, p.sess.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()

	inA := p.transfers(p.A.ID, p.sess)
	if tr := inA["forged"]; tr == nil || tr["status"] != "pending" || tr["review"] != "missing_remote" || tr["verified"] != false {
		t.Fatalf("forged transfer in A: %v", tr)
	}
	if tr := inA[real]; tr["status"] != "pending" || tr["verified"] != false {
		t.Fatalf("raised version in A: %v", tr)
	}
	if _, ok := inA["foreign"]; ok {
		t.Fatal("a record of other spaces survived the restore")
	}
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transfers/forged/reject"), nil, p.sess.Token, p.sess.CSRF), 422, "push forged")
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transfers/"+real+"/reject"), nil, p.sess.Token, p.sess.CSRF), 422, "push raised version")
	if err := p.a.s.transfers.SyncAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.transfers(p.B.ID, p.sess)["forged"]; ok || p.balance(p.B.ID, p.accB) != bBefore {
		t.Fatal("the forged backup reached B")
	}
}

// P33: restoring a server backup restores every space consistently; nothing
// waits for review.
func TestP33ServerRestoreNeedsNoReview(t *testing.T) {
	p := newPair(t)
	admin := p.a.issue(p.a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	id := p.send("3", "3")
	b := createdBackup(t, p.a.do("POST", "/api/backups", map[string]any{}, admin.Token, admin.CSRF))
	status(t, p.a.do("PATCH", spacePath(p.A.ID, "/transfers/"+id), map[string]any{"from_amount": "9"}, p.sess.Token, p.sess.CSRF), 200, "change")
	res := p.a.do("POST", "/api/backups/"+b["filename"].(string)+"/restore", nil, admin.Token, admin.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	for _, sp := range []int64{p.A.ID, p.B.ID} {
		if tr := p.transfers(sp, p.sess)[id]; tr["status"] != "ok" || tr["from"].(map[string]any)["amount"] != "3" {
			t.Fatalf("space %d %v", sp, tr)
		}
	}
}

// P54: restoring a space while transfers and a server snapshot run does not
// deadlock.
func TestP54RestoreTransfersAndSnapshotConcurrently(t *testing.T) {
	p := newPair(t)
	admin := p.a.issue(p.a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	backup := p.backupA()
	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(3)
			go func() {
				defer wg.Done()
				res := p.a.do("POST", spacePath(p.A.ID, "/transfers"), map[string]any{
					"to_space_id": p.B.ID, "from_account_id": p.accA, "to_account_id": p.accB, "from_amount": "1", "to_amount": "1", "date": "2026-10-01",
				}, p.sess.Token, p.sess.CSRF)
				res.Body.Close()
			}()
			go func() {
				defer wg.Done()
				res := p.a.do("POST", spacePath(p.A.ID, "/backups/"+backup+"/restore"), nil, p.sess.Token, p.sess.CSRF)
				res.Body.Close()
			}()
			go func() {
				defer wg.Done()
				res := p.a.do("POST", "/api/backups", map[string]any{}, admin.Token, admin.CSRF)
				res.Body.Close()
			}()
		}
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(90 * time.Second):
		t.Fatal("deadlock")
	}
}

// P45: a server backup carries the signing key, so on a server that had
// another key the restored transfers still verify and stay linked.
func TestP45ServerMovesWithItsKey(t *testing.T) {
	p := newPair(t)
	admin := p.a.issue(p.a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	id := p.send("6", "6")
	b := createdBackup(t, p.a.do("POST", "/api/backups", map[string]any{}, admin.Token, admin.CSRF))
	content, _ := os.ReadFile(p.a.s.backups.ServerPath(b["filename"].(string)))

	fresh := newTestApp(t)
	if fresh.s.keys.Key().KID() == p.a.s.keys.Key().KID() {
		t.Fatal("the test servers share a key")
	}
	fa := fresh.issue(fresh.createUser("admin@new.test", "secret1", auth.RoleAdmin), false)
	up := createdBackup(t, fresh.upload("/api/backups/upload", "server.zip", content, nil, fa))
	res := fresh.do("POST", "/api/backups/"+up["filename"].(string)+"/restore", nil, fa.Token, fa.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	if fresh.s.keys.Key().KID() != p.a.s.keys.Key().KID() {
		t.Fatal("the restored server does not use the backup's key")
	}
	list, err := fresh.s.transfers.List(context.Background(), p.A.ID)
	if err != nil || len(list) != 1 || list[0].UUID != id || !list[0].Verified || list[0].Frozen {
		t.Fatalf("transfers after the move %+v %v", list, err)
	}
}

// P26: after A is restored, A's ids count up again, so a new transaction can
// take the id the transfer's side had. The side is tied to the transfer by
// uuid, never by id: the new transaction stays an ordinary one, and the
// transfer comes back with a side of its own.
func TestP26ReusedIDIsNotTheTransfer(t *testing.T) {
	p := newPair(t)
	backup := p.backupA()
	id := p.send("5", "5")
	sideID := decodeJSON(t, p.a.do("GET", spacePath(p.A.ID, "/transactions"), nil, p.sess.Token, ""))["data"].([]any)[0].(map[string]any)["id"]
	// Unlinked, the restore has nothing to review and A forgets the transfer.
	status(t, p.a.do("DELETE", spacePath(p.A.ID, "/links/"+strconv.FormatInt(p.B.ID, 10)), nil, p.sess.Token, p.sess.CSRF), 204, "unlink")
	p.restoreA(backup)
	created := p.post(p.A.ID, "/transactions", map[string]any{"type": "expense", "account_id": p.accA, "amount": 1, "date": "2026-10-01"})
	newID := created["data"].(map[string]any)["id"]
	if newID != sideID {
		t.Fatalf("the scenario needs the id reused: side %v, new %v", sideID, newID)
	}
	status(t, p.a.do("POST", spacePath(p.A.ID, "/links"), map[string]any{"target_space_id": p.B.ID}, p.sess.Token, p.sess.CSRF), 200, "relink")

	path := spacePath(p.A.ID, "/transactions/"+strconv.FormatFloat(newID.(float64), 'f', -1, 64))
	status(t, p.a.do("PATCH", path, map[string]any{"type": "expense", "account_id": p.accA, "amount": 2, "date": "2026-10-01"}, p.sess.Token, p.sess.CSRF), 200, "the new transaction is ordinary")
	if tr := p.transfers(p.A.ID, p.sess)[id]; tr == nil || tr["status"] != "ok" {
		t.Fatalf("the transfer did not come back: %v", tr)
	}
	if b := p.balance(p.A.ID, p.accA); b != "-7" {
		t.Fatalf("A balance %s, want the expense (2) and the transfer (5)", b)
	}
}
