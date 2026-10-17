package uploads

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	authrepo "savvy-go/internal/db/repo/auth"
	repo "savvy-go/internal/db/repo/uploads"
)

type part = struct {
	Number int
	ETag   string
}

func newService(t *testing.T) (Service, int64) {
	t.Helper()
	d := dbtest.Server(t)
	uid, err := authrepo.Repo{DB: d}.InsertUser(context.Background(), "Ann", "ann@test.com", "h", "user", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return Service{DB: d, Root: t.TempDir(), SignSecret: "secret"}, uid
}

func ptr[T any](v T) *T { return &v }

func create(t *testing.T, s Service, uid int64, name string, size int64) *Upload {
	t.Helper()
	u, err := s.Create(context.Background(), uid, "transaction-imports", name, &size, nil)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestCreateRefusesWhatTheBucketDoesNotTake(t *testing.T) {
	s, uid := newService(t)
	ctx := context.Background()
	tests := []struct {
		name     string
		bucket   string
		filename string
		size     int64
		mime     *string
	}{
		{"an unknown bucket", "avatars", "a.csv", 10, nil},
		{"an executable", "transaction-imports", "a.exe", 10, nil},
		{"no extension", "transaction-imports", "a", 10, nil},
		{"an empty file", "transaction-imports", "a.csv", 0, nil},
		{"a file over the limit", "transaction-imports", "a.csv", 512*1024*1024 + 1, nil},
		{"a foreign content type", "transaction-imports", "a.csv", 10, ptr("image/png")},
	}
	for _, tc := range tests {
		size := tc.size
		if u, err := s.Create(ctx, uid, tc.bucket, tc.filename, &size, tc.mime); err == nil || u != nil {
			t.Errorf("%s was accepted", tc.name)
		}
	}
	rows, err := s.UsageByBucket(ctx)
	if err != nil || len(rows) != 0 {
		t.Errorf("refused uploads leave no record: %v, %v", rows, err)
	}
}

func TestCreatePlansTheParts(t *testing.T) {
	s, uid := newService(t)
	ctx := context.Background()

	small := create(t, s, uid, "Bank Export (2024).CSV", 20*1024*1024)
	if small.PartSize != defaultPartSize || small.TotalParts == nil || *small.TotalParts != 3 {
		t.Errorf("20 MiB in 8 MiB parts: size %d, parts %v", small.PartSize, small.TotalParts)
	}
	if small.Status != uploadPending || small.ExpiresAt == nil || !small.ExpiresAt.After(time.Now()) {
		t.Errorf("a new upload is pending with a deadline: %+v", small)
	}
	if !strings.HasPrefix(small.ObjectKey, "transaction-imports/") || !strings.HasSuffix(small.ObjectKey, "/"+small.ID+"/bank-export-2024.csv") {
		t.Errorf("object key %q", small.ObjectKey)
	}

	one := create(t, s, uid, "a.csv", 1)
	if one.TotalParts == nil || *one.TotalParts != 1 {
		t.Errorf("one byte is one part: %v", one.TotalParts)
	}

	unknown, err := s.Create(ctx, uid, "transaction-imports", "a.csv", nil, nil)
	if err != nil || unknown.TotalParts != nil || unknown.Size != nil {
		t.Errorf("an unknown size has no part count: %+v, %v", unknown, err)
	}
	if got, err := s.ByID(ctx, "missing"); got != nil || err != nil {
		t.Errorf("ByID of an unknown id: %v, %v", got, err)
	}
}

func TestPartURLsAreSignedAndExpire(t *testing.T) {
	s, uid := newService(t)
	u := create(t, s, uid, "a.csv", 10)

	signed, err := s.SignPart(u, 2)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := signed["url"].(string)
	if !strings.HasPrefix(raw, "/api/uploads/"+u.ID+"/parts/2?") {
		t.Fatalf("url %q", raw)
	}
	expires := time.Now().Add(time.Hour).Unix()
	sig := s.sign(u.ID, 2, expires)
	if !s.VerifyPart(u.ID, 2, expires, sig) {
		t.Error("a signature made by the service must verify")
	}
	for name, ok := range map[string]bool{
		"another part":     s.VerifyPart(u.ID, 3, expires, sig),
		"another upload":   s.VerifyPart("other", 2, expires, sig),
		"another deadline": s.VerifyPart(u.ID, 2, expires+1, sig),
		"a blank":          s.VerifyPart(u.ID, 2, expires, ""),
		"a past deadline":  s.VerifyPart(u.ID, 2, time.Now().Add(-time.Second).Unix(), s.sign(u.ID, 2, time.Now().Add(-time.Second).Unix())),
		"another secret":   Service{SignSecret: "other"}.VerifyPart(u.ID, 2, expires, sig),
	} {
		if ok {
			t.Errorf("%s verified", name)
		}
	}

	for _, n := range []int{0, -1, maxParts + 1} {
		if _, err := s.SignPart(u, n); err == nil {
			t.Errorf("part %d was signed", n)
		}
	}
	done := *u
	done.Status = uploadCompleted
	if _, err := s.SignPart(&done, 1); err == nil {
		t.Error("a finished upload takes no more parts")
	}
}

func TestPartsAssembleIntoTheFileInOrder(t *testing.T) {
	s, uid := newService(t)
	ctx := context.Background()
	u := create(t, s, uid, "a.csv", 11)

	if parts := s.ListParts(u); len(parts) != 0 {
		t.Fatalf("parts before any arrive: %v", parts)
	}
	etag2, n2, err := s.StorePart(u, 2, strings.NewReader("world"))
	if err != nil || n2 != 5 {
		t.Fatalf("store part 2: %d, %v", n2, err)
	}
	etag1, _, err := s.StorePart(u, 1, strings.NewReader("hello "))
	if err != nil {
		t.Fatal(err)
	}
	// Storing a part again replaces it.
	if _, _, err := s.StorePart(u, 2, strings.NewReader("world")); err != nil {
		t.Fatal(err)
	}

	listed := s.ListParts(u)
	if len(listed) != 2 || listed[0]["PartNumber"] != 1 || listed[1]["PartNumber"] != 2 {
		t.Fatalf("parts must be listed in order: %v", listed)
	}
	if listed[0]["ETag"] != etag1 || listed[1]["ETag"] != etag2 || listed[1]["Size"] != int64(5) {
		t.Errorf("part details %v (etags %s %s)", listed, etag1, etag2)
	}

	// The client may list parts in any order; the file is assembled by number.
	done, err := s.Complete(ctx, u, []part{{2, etag2}, {1, etag1}})
	if err != nil {
		t.Fatal(err)
	}
	if !done.Completed() || done.Path == nil {
		t.Fatalf("completed upload %+v", done)
	}
	body, err := s.ReadFile(done)
	if err != nil || string(body) != "hello world" {
		t.Errorf("file %q, %v", body, err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, u.ID, "parts")); !os.IsNotExist(err) {
		t.Errorf("the parts are removed once assembled: %v", err)
	}
	if again, err := s.Complete(ctx, done, []part{{1, etag1}}); err != nil || again != done {
		t.Errorf("completing twice is a no-op: %v", err)
	}
	if loc := s.Location(done); !strings.HasPrefix(loc, "/api/s3/transaction-imports/") || strings.Contains(loc, "//") {
		t.Errorf("location %q", loc)
	}
	if _, _, err := s.StorePart(done, 3, strings.NewReader("x")); err == nil {
		t.Error("a finished upload takes no more parts")
	}
}

func TestCompleteNeedsEveryPartItNames(t *testing.T) {
	s, uid := newService(t)
	ctx := context.Background()
	u := create(t, s, uid, "a.csv", 10)
	if _, _, err := s.StorePart(u, 1, strings.NewReader("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Complete(ctx, u, nil); err == nil {
		t.Error("completing without parts must fail")
	}
	if _, err := s.Complete(ctx, u, []part{{1, ""}, {2, ""}}); err == nil {
		t.Error("completing with a part that never arrived must fail")
	}
	if got, _ := s.ByID(ctx, u.ID); got.Status != uploadPending {
		t.Errorf("a failed completion leaves the upload pending: %s", got.Status)
	}
}

func TestReadFileOnlyOnceCompleted(t *testing.T) {
	s, uid := newService(t)
	u := create(t, s, uid, "a.csv", 3)
	if _, err := s.ReadFile(u); err == nil {
		t.Error("a pending upload has no file to read")
	}
}

func TestAbortRemovesTheFilesAndMarksTheUpload(t *testing.T) {
	s, uid := newService(t)
	ctx := context.Background()
	u := create(t, s, uid, "a.csv", 3)
	if _, _, err := s.StorePart(u, 1, strings.NewReader("abc")); err != nil {
		t.Fatal(err)
	}
	if err := s.Abort(ctx, u); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, u.ID)); !os.IsNotExist(err) {
		t.Errorf("parts remain after an abort: %v", err)
	}
	got, _ := s.ByID(ctx, u.ID)
	if got.Status != uploadAborted {
		t.Errorf("status %s", got.Status)
	}
	if _, err := s.Complete(ctx, got, []part{{1, ""}}); err == nil {
		t.Error("an aborted upload cannot be completed")
	}
}

func TestDiscardDeletesTheAssembledFile(t *testing.T) {
	s, uid := newService(t)
	ctx := context.Background()
	u := create(t, s, uid, "a.csv", 3)
	etag, _, _ := s.StorePart(u, 1, strings.NewReader("abc"))
	done, err := s.Complete(ctx, u, []part{{1, etag}})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(s.Root, filepath.FromSlash(*done.Path))
	if _, err := os.Stat(file); err != nil {
		t.Fatal(err)
	}
	if err := s.Discard(ctx, done); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Errorf("the file outlives its import: %v", err)
	}
	if got, _ := s.ByID(ctx, u.ID); got.Status != uploadConsumed {
		t.Errorf("status %s", got.Status)
	}
}

func TestPruneExpiredAbortsOnlyStalePendingUploads(t *testing.T) {
	s, uid := newService(t)
	ctx := context.Background()
	fresh := create(t, s, uid, "fresh.csv", 3)

	now := time.Now().UTC()
	r := repo.Repo{DB: s.DB}
	for id, exp := range map[string]time.Time{"stale": now.Add(-time.Hour), "stale-done": now.Add(-time.Hour)} {
		if err := r.Insert(ctx, repo.New{
			ID: id, UserID: uid, Bucket: "transaction-imports", ObjectKey: "transaction-imports/" + id, Disk: "local",
			OriginalName: id + ".csv", Size: ptr(int64(3)), PartSize: 1024, Status: uploadPending, ExpiresAt: exp, Now: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.SetStatus(ctx, "stale-done", uploadCompleted, now); err != nil {
		t.Fatal(err)
	}
	staleParts := filepath.Join(s.Root, "stale", "parts")
	if err := os.MkdirAll(staleParts, 0o775); err != nil {
		t.Fatal(err)
	}

	if err := s.PruneExpired(ctx); err != nil {
		t.Fatal(err)
	}
	status := func(id string) string {
		u, err := s.ByID(ctx, id)
		if err != nil || u == nil {
			t.Fatalf("%s: %v", id, err)
		}
		return u.Status
	}
	if got := status("stale"); got != uploadAborted {
		t.Errorf("stale upload %s", got)
	}
	if got := status("stale-done"); got != uploadCompleted {
		t.Errorf("a completed upload is not pruned, got %s", got)
	}
	if got := status(fresh.ID); got != uploadPending {
		t.Errorf("a live upload is kept, got %s", got)
	}
	if _, err := os.Stat(staleParts); !os.IsNotExist(err) {
		t.Errorf("stale parts remain: %v", err)
	}
}

func TestObjectKeysAreSafeFileNames(t *testing.T) {
	for in, want := range map[string]string{
		"Bank Export.csv":      "bank-export.csv",
		"../../etc/passwd.txt": "passwd.txt",
		"счёт.csv":             "file.csv",
		"a b  c.CSV":           "a-b--c.csv",
	} {
		key := buildObjectKey("/prefix/", "id1", in)
		if !strings.HasPrefix(key, "prefix/") || !strings.HasSuffix(key, "/id1/"+want) || strings.Contains(key, "..") {
			t.Errorf("%q -> %q, want …/id1/%s", in, key, want)
		}
	}
}

func TestRandomSecretsDiffer(t *testing.T) {
	a, err := randomSecret()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := randomSecret()
	if len(a) != 64 || a == b {
		t.Errorf("secrets %q %q", a, b)
	}
}
