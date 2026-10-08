package uploads_test

import (
	"context"
	"testing"
	"time"

	"savvy-go/internal/db/dbtest"
	authrepo "savvy-go/internal/db/repo/auth"
	uploadsrepo "savvy-go/internal/db/repo/uploads"
)

// newRepo opens a server database with one user, who owns every upload.
func newRepo(t *testing.T) (uploadsrepo.Repo, int64) {
	t.Helper()
	d := dbtest.Server(t)
	uid, err := authrepo.Repo{DB: d}.InsertUser(context.Background(), "Ann", "ann@test.com", "h", "user", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return uploadsrepo.Repo{DB: d}, uid
}

func newUpload(user int64, id, bucket string, size int64, expires time.Time) uploadsrepo.New {
	mime := "text/csv"
	return uploadsrepo.New{
		UserID: user, ID: id, Bucket: bucket, ObjectKey: bucket + "/" + id, Disk: "local", OriginalName: id + ".csv",
		Status: "pending", MimeType: &mime, Size: &size, PartSize: 1024, ExpiresAt: expires, Now: time.Now().UTC(),
	}
}

func TestUploadRoundTripKeepsOptionalFields(t *testing.T) {
	ctx := context.Background()
	r, user := newRepo(t)
	if u, err := r.ByID(ctx, "nope"); u != nil || err != nil {
		t.Fatalf("unknown upload = %v, %v", u, err)
	}
	expires := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	in := newUpload(user, "u1", "imports", 5000, expires)
	parts := 5
	in.TotalParts = &parts
	if err := r.Insert(ctx, in); err != nil {
		t.Fatal(err)
	}
	u, _ := r.ByID(ctx, "u1")
	if u == nil || u.Status != "pending" || u.Size == nil || *u.Size != 5000 || u.PartSize != 1024 ||
		u.TotalParts == nil || *u.TotalParts != 5 || u.Path != nil || u.ExpiresAt == nil || !u.ExpiresAt.Equal(expires) {
		t.Fatalf("upload = %+v", u)
	}

	if err := r.Complete(ctx, "u1", "imports/u1", "completed", 5, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if u, _ := r.ByID(ctx, "u1"); u.Status != "completed" || u.Path == nil || *u.Path != "imports/u1" {
		t.Fatalf("after Complete: %+v", u)
	}
}

func TestOnlyPendingUploadsPastTheirTimeExpire(t *testing.T) {
	ctx := context.Background()
	r, user := newRepo(t)
	now := time.Now().UTC()
	_ = r.Insert(ctx, newUpload(user, "old", "b", 1, now.Add(-time.Hour)))
	_ = r.Insert(ctx, newUpload(user, "fresh", "b", 1, now.Add(time.Hour)))
	_ = r.Insert(ctx, newUpload(user, "done", "b", 1, now.Add(-time.Hour)))
	_ = r.SetStatus(ctx, "done", "completed", now)

	ids, err := r.ExpiredPending(ctx, now)
	if err != nil || len(ids) != 1 || ids[0] != "old" {
		t.Fatalf("expired = %v, %v", ids, err)
	}
}

func TestUsageSumsBytesPerBucketAndStatus(t *testing.T) {
	ctx := context.Background()
	r, user := newRepo(t)
	now := time.Now().UTC().Add(time.Hour)
	_ = r.Insert(ctx, newUpload(user, "a", "imports", 100, now))
	_ = r.Insert(ctx, newUpload(user, "b", "imports", 250, now))
	_ = r.SetStatus(ctx, "b", "completed", now)

	usage, err := r.UsageByBucket(ctx)
	if err != nil || len(usage) != 2 {
		t.Fatalf("usage = %+v, %v", usage, err)
	}
	got := map[string]int64{}
	for _, u := range usage {
		got[u.Status] = u.Bytes
		if u.Bucket != "imports" || u.Count != 1 {
			t.Fatalf("row = %+v", u)
		}
	}
	if got["pending"] != 100 || got["completed"] != 250 {
		t.Fatalf("bytes = %v", got)
	}
}
