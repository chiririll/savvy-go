package httpserver

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"savvy-go/internal/auth"
	"savvy-go/internal/db"
	"savvy-go/internal/spaces"
)

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

// After A is restored, every difference with B is pending in A and B
// does not change until someone decides; pending changes stay out of A's
// balance, which matches the backup.
func TestRestoreReviewsTransfers(t *testing.T) {
	p := newPair(t)
	kept := p.send("1", "1")
	changed := p.send("2", "2")
	deleted := p.send("4", "4")
	backup := p.backupA()
	balanceAtBackup := p.balance(p.A.ID, p.accA)

	created := p.send("8", "8")
	status(
		t,
		p.a.do(
			"PATCH",
			spacePath(p.B.ID, "/transfers/"+changed),
			map[string]any{"from_amount": "20", "to_amount": "20"},
			p.sess.Token,
			p.sess.CSRF,
		),
		200,
		"change",
	)
	status(t, p.a.do("DELETE", spacePath(p.B.ID, "/transfers/"+deleted), nil, p.sess.Token, p.sess.CSRF), 204, "delete")
	bBefore := p.transfers(p.B.ID, p.sess)
	bBalance := p.balance(p.B.ID, p.accB)

	p.restoreA(backup)

	inA := p.transfers(p.A.ID, p.sess)
	for id, review := range map[string]string{created: "created_remote", changed: "changed_remote",
		deleted: "deleted_remote"} {
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
	_ = p.a.svc.Spaces.SetMember(context.Background(), p.A.ID, viewer, spaces.RoleEditor)
	vs := p.a.issue(viewer, false)
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/transfers/"+changed+"/reject"), nil, vs.Token, vs.CSRF),
		403,
		"reject without rights in B",
	)

	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/transfers/"+created+"/accept"), nil, p.sess.Token, p.sess.CSRF),
		200,
		"accept created",
	)
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/transfers/"+deleted+"/accept"), nil, p.sess.Token, p.sess.CSRF),
		204,
		"accept deleted",
	)
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/transfers/"+changed+"/reject"), nil, p.sess.Token, p.sess.CSRF),
		200,
		"reject changed",
	)

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

// A crash before the review merge finishes reviews again at startup
// instead of merging automatically.
func TestReviewMarkSurvivesCrash(t *testing.T) {
	p := newPair(t)
	backup := p.backupA()
	created := p.send("8", "8")
	ctx := context.Background()
	space, _ := p.a.svc.Spaces.Get(ctx, p.A.ID)

	// Restore without the review merge, as if the process died right after
	// the swap: the mark is in the file.
	b := p.a.svc.Backups
	b.Transfers = nil
	if err := b.RestoreSpace(ctx, *space, backup, 0); err != nil {
		t.Fatal(err)
	}
	if err := p.a.svc.Transfers.SyncAll(ctx); err != nil {
		t.Fatal(err)
	}
	if tr := p.transfers(p.A.ID, p.sess)[created]; tr == nil || tr["status"] != "pending" {
		t.Fatalf("startup merged automatically: %v", tr)
	}
}

// A transfer whose account is not in the restored space needs attention
// until another account is chosen.
func TestMissingAccountNeedsAttention(t *testing.T) {
	p := newPair(t)
	backup := p.backupA()
	later := p.account(p.A.ID, "GBP")
	body := p.post(p.A.ID, "/transfers", map[string]any{
		"to_space_id": p.B.ID, "from_account_id": later, "to_account_id": p.accB, "from_amount": "7", "to_amount": "7",
		"date": "2026-10-01",
	})
	id := body["data"].(map[string]any)["uuid"].(string)
	p.restoreA(backup)
	status(t, p.a.do("POST", spacePath(p.A.ID, "/transfers/"+id+"/accept"), nil, p.sess.Token, p.sess.CSRF), 200, "accept")
	if tr := p.transfers(p.A.ID, p.sess)[id]; tr["status"] != "needs_attention" {
		t.Fatalf("status %v", tr)
	}
	status(
		t,
		p.a.do(
			"PATCH",
			spacePath(p.A.ID, "/transfers/"+id),
			map[string]any{"from_account_id": p.accA},
			p.sess.Token,
			p.sess.CSRF,
		),
		200,
		"choose account",
	)
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
	_, _ = mw.Write(
		[]byte(`{"format":1,"kind":"space","store":"sqlite","files":{"space.sqlite":"` + sum + `"},"space_uuid":"x"}`),
	)
	_ = zw.Close()
	return buf.Bytes()
}

// A doctored backup cannot reach the other space: a
// made-up transfer and a raised version stay pending in A with no way to
// push them, and a genuine record of other spaces is dropped.
func TestForgedBackupCannotReachOtherSpace(t *testing.T) {
	p := newPair(t)
	real := p.send("5", "5")
	backup := p.backupA()
	uuidA, uuidB := p.A.UUID, p.B.UUID
	forged := forgeBackup(t, p.a.svc.Backups.SpacePath(p.A.ID, backup),
		`UPDATE space_transfers SET version = 99, from_amount = 1 WHERE uuid = '`+real+`'`,
		`INSERT INTO space_transfers (uuid, from_space_uuid, from_account_id, from_amount, from_currency, from_decimals,
			to_space_uuid, to_account_id, to_amount, to_currency, to_decimals, date, version, signer_kid, signature)
			VALUES ('forged', '`+uuidA+`', 1, 100000, 'EUR', 2, '`+uuidB+`', 1, 100000, 'USD', 2,
				'2026-10-01', 1, 'nobody', 'AAAA')`,
		`INSERT INTO space_transfers (uuid, from_space_uuid, from_account_id, from_amount, from_currency, from_decimals,
			to_space_uuid, to_account_id, to_amount, to_currency, to_decimals, date, version, signer_kid, signature)
			VALUES ('foreign', 'someone', 1, 1, 'EUR', 2, '`+uuidB+`', 1, 1, 'USD', 2, '2026-10-01', 1, 'nobody', 'AAAA')`,
	)
	up := createdBackup(t, p.a.upload(spacePath(p.A.ID, "/backups/upload"), "forged.zip", forged, nil, p.sess))
	// The manifest names another uuid: restore checks the database, not it.
	bBefore := p.balance(p.B.ID, p.accB)
	res := p.a.do(
		"POST",
		spacePath(p.A.ID, "/backups/"+up["filename"].(string)+"/restore"),
		nil,
		p.sess.Token,
		p.sess.CSRF,
	)
	if res.StatusCode != 200 {
		t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()

	inA := p.transfers(p.A.ID, p.sess)
	if tr := inA["forged"]; tr == nil || tr["status"] != "pending" || tr["review"] != "missing_remote" ||
		tr["verified"] != false {
		t.Fatalf("forged transfer in A: %v", tr)
	}
	if tr := inA[real]; tr["status"] != "pending" || tr["verified"] != false {
		t.Fatalf("raised version in A: %v", tr)
	}
	if _, ok := inA["foreign"]; ok {
		t.Fatal("a record of other spaces survived the restore")
	}
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/transfers/forged/reject"), nil, p.sess.Token, p.sess.CSRF),
		422,
		"push forged",
	)
	status(
		t,
		p.a.do("POST", spacePath(p.A.ID, "/transfers/"+real+"/reject"), nil, p.sess.Token, p.sess.CSRF),
		422,
		"push raised version",
	)
	if err := p.a.svc.Transfers.SyncAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.transfers(p.B.ID, p.sess)["forged"]; ok || p.balance(p.B.ID, p.accB) != bBefore {
		t.Fatal("the forged backup reached B")
	}
}

// Restoring a server backup restores every space consistently; nothing
// waits for review.
func TestServerRestoreNeedsNoReview(t *testing.T) {
	p := newPair(t)
	admin := p.a.issue(p.a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	id := p.send("3", "3")
	b := createdBackup(t, p.a.do("POST", "/api/backups", map[string]any{}, admin.Token, admin.CSRF))
	status(
		t,
		p.a.do("PATCH", spacePath(p.A.ID, "/transfers/"+id), map[string]any{"from_amount": "9"}, p.sess.Token, p.sess.CSRF),
		200,
		"change",
	)
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

// Restoring a space while transfers and a server snapshot run does not
// deadlock.
func TestRestoreTransfersSnapshotConcurrentlyNotDeadlock(t *testing.T) {
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
					"to_space_id": p.B.ID, "from_account_id": p.accA, "to_account_id": p.accB, "from_amount": "1", "to_amount": "1",
					"date": "2026-10-01",
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

// A server backup carries the signing key, so on a server that had
// another key the restored transfers still verify and stay linked.
func TestServerBackupCarriesItsKey(t *testing.T) {
	p := newPair(t)
	admin := p.a.issue(p.a.createUser("admin@test.com", "secret1", auth.RoleAdmin), false)
	id := p.send("6", "6")
	b := createdBackup(t, p.a.do("POST", "/api/backups", map[string]any{}, admin.Token, admin.CSRF))
	content, _ := os.ReadFile(p.a.svc.Backups.ServerPath(b["filename"].(string)))

	fresh := newTestApp(t)
	if fresh.svc.Keys.Key().KID() == p.a.svc.Keys.Key().KID() {
		t.Fatal("the test servers share a key")
	}
	fa := fresh.issue(fresh.createUser("admin@new.test", "secret1", auth.RoleAdmin), false)
	up := createdBackup(t, fresh.upload("/api/backups/upload", "server.zip", content, nil, fa))
	res := fresh.do("POST", "/api/backups/"+up["filename"].(string)+"/restore", nil, fa.Token, fa.CSRF)
	if res.StatusCode != 200 {
		t.Fatalf("restore %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
	if fresh.svc.Keys.Key().KID() != p.a.svc.Keys.Key().KID() {
		t.Fatal("the restored server does not use the backup's key")
	}
	list, err := fresh.svc.Transfers.List(context.Background(), p.A.ID)
	if err != nil || len(list) != 1 || list[0].UUID != id || !list[0].Verified || list[0].Frozen {
		t.Fatalf("transfers after the move %+v %v", list, err)
	}
}
