package httpserver

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/chiririll/savvy-plus/internal/auth"
)

func TestMultipartImportPipeline(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("imp@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	accID, _ := seedMoney(t, a, sess)

	csv := "Date,Amount,Description\n2024-01-01,-12.50,Coffee\n2024-01-02,2000,Salary\n"

	res := a.do("POST", "/api/s3/multipart", map[string]any{
		"bucket": "transaction-imports", "filename": "statement.csv", "type": "text/csv", "size": len(csv),
	}, sess.Token, sess.CSRF)
	create := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("create %d %v", res.StatusCode, create)
	}
	uploadID := create["uploadId"].(string)
	key := create["key"].(string)
	if !strings.Contains(key, "transaction-imports/") {
		t.Fatalf("key %s", key)
	}

	res = a.do("GET", "/api/s3/multipart/"+uploadID+"/1?key="+url.QueryEscape(key), nil, sess.Token, "")
	sign := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("sign %d %v", res.StatusCode, sign)
	}
	signed, err := url.Parse(sign["url"].(string))
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PUT", a.srv.URL+signed.RequestURI(), bytes.NewReader([]byte(csv)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	put, err := a.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if put.StatusCode != 200 {
		t.Fatalf("put %d %v", put.StatusCode, decodeJSON(t, put))
	}
	etag := strings.Trim(put.Header.Get("ETag"), `"`)
	io.Copy(io.Discard, put.Body)
	put.Body.Close()

	res = a.do("POST", "/api/s3/multipart/"+uploadID+"/complete?key="+url.QueryEscape(key), map[string]any{
		"parts": []map[string]any{{"PartNumber": 1, "ETag": etag}},
	}, sess.Token, sess.CSRF)
	complete := decodeJSON(t, res)
	if res.StatusCode != 200 || complete["location"] == nil {
		t.Fatalf("complete %d %v", res.StatusCode, complete)
	}

	res = a.do("POST", "/api/transactions/import/parse", map[string]any{"upload_id": uploadID}, sess.Token, sess.CSRF)
	parsed := decodeJSON(t, res)
	if res.StatusCode != 200 {
		t.Fatalf("parse %d %v", res.StatusCode, parsed)
	}
	importID := parsed["data"].(map[string]any)["import_id"].(string)

	res = a.do("GET", "/api/transactions/import/"+importID, nil, sess.Token, "")
	status := decodeJSON(t, res)["data"].(map[string]any)
	if res.StatusCode != 200 || status["status"] != "parsed" {
		t.Fatalf("parsed status %d %v", res.StatusCode, status)
	}
	if int(status["total_rows"].(float64)) != 2 {
		t.Fatalf("rows %v", status)
	}
	headers := status["parse"].(map[string]any)["headers"].([]any)
	if len(headers) != 3 {
		t.Fatalf("headers %v", headers)
	}

	res = a.do("POST", "/api/transactions/import/execute", map[string]any{
		"import_id": importID,
		"mapping":   map[string]any{"date": 0, "amount": 1, "description": 2},
		"options": map[string]any{
			"date_format": "ISO", "amount_format": "US",
			"default_account_id": accID, "default_type": "expense",
		},
	}, sess.Token, sess.CSRF)
	exec := decodeJSON(t, res)
	if res.StatusCode != 200 || exec["data"].(map[string]any)["status"] != "importing" {
		t.Fatalf("execute %d %v", res.StatusCode, exec)
	}

	res = a.do("GET", "/api/transactions/import/"+importID, nil, sess.Token, "")
	final := decodeJSON(t, res)["data"].(map[string]any)
	if final["status"] != "completed" {
		t.Fatalf("final %v", final)
	}
	if int(final["result"].(map[string]any)["created"].(float64)) != 2 {
		t.Fatalf("created %v", final)
	}

	res = a.do("GET", "/api/transactions", nil, sess.Token, "")
	list := decodeJSON(t, res)["data"].([]any)
	if len(list) != 2 {
		t.Fatalf("txs %d", len(list))
	}
}

func TestUploadUnknownBucket(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("bkt@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	res := a.do("POST", "/api/s3/multipart", map[string]any{
		"bucket": "nope", "filename": "x.csv", "size": 10,
	}, sess.Token, sess.CSRF)
	if res.StatusCode != 422 {
		t.Fatalf("want 422 got %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
}

func TestUploadSignForbiddenForOtherUser(t *testing.T) {
	a := newTestApp(t)
	u1 := a.createUser("o1@test.com", "secret1", auth.RoleReadWrite)
	u2 := a.createUser("o2@test.com", "secret1", auth.RoleReadWrite)
	s1 := a.issue(u1, false)
	s2 := a.issue(u2, false)
	res := a.do("POST", "/api/s3/multipart", map[string]any{
		"bucket": "transaction-imports", "filename": "statement.csv", "size": 100,
	}, s1.Token, s1.CSRF)
	create := decodeJSON(t, res)
	res = a.do("GET", "/api/s3/multipart/"+create["uploadId"].(string)+"/1?key="+url.QueryEscape(create["key"].(string)), nil, s2.Token, "")
	if res.StatusCode != 403 {
		t.Fatalf("want 403 got %d %v", res.StatusCode, decodeJSON(t, res))
	}
	res.Body.Close()
}

func TestBackupCreateAndList(t *testing.T) {
	a := newTestApp(t)
	u := a.createUser("bk@test.com", "secret1", auth.RoleReadWrite)
	sess := a.issue(u, false)
	res := a.do("POST", "/api/backups", map[string]any{"note": "nightly"}, sess.Token, sess.CSRF)
	body := decodeJSON(t, res)
	if res.StatusCode != 201 {
		t.Fatalf("create %d %v", res.StatusCode, body)
	}
	res = a.do("GET", "/api/backups", nil, sess.Token, "")
	list := decodeJSON(t, res)
	if res.StatusCode != 200 || len(list["data"].([]any)) != 1 {
		t.Fatalf("list %d %v", res.StatusCode, list)
	}
}
