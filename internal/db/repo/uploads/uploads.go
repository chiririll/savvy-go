// Package uploadsrepo stores the bookkeeping of chunked uploads (the files
// themselves live on disk, managed by package uploads).
package uploads

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Repo reads and writes the uploads table of the server database.
type Repo struct{ DB store.DB }

// Upload is one row.
type Upload struct {
	ID, Bucket, ObjectKey, Disk, OriginalName, Status string
	UserID                                            *int64
	Path, MimeType                                    *string
	Size                                              *int64
	PartSize                                          int
	TotalParts                                        *int
	ExpiresAt                                         *time.Time
}

// New is a pending upload.
type New struct {
	ID, Bucket, ObjectKey, Disk, OriginalName, Status string
	UserID                                            int64
	MimeType                                          *string
	Size                                              *int64
	PartSize                                          int
	TotalParts                                        *int
	ExpiresAt, Now                                    time.Time
}

func (r Repo) Insert(ctx context.Context, u New) error {
	var parts db.NInt
	if u.TotalParts != nil {
		parts = db.NI(int64(*u.TotalParts))
	}
	return db.Q(r.DB).InsertUpload(ctx, sqlc.InsertUploadParams{
		ID: u.ID, UserID: db.NI(u.UserID), Bucket: u.Bucket, ObjectKey: u.ObjectKey, Disk: u.Disk,
		OriginalName: u.OriginalName, MimeType: db.NullString(u.MimeType), Size: db.NullInt64(u.Size),
		PartSize: db.NI(int64(u.PartSize)), TotalParts: parts, Status: u.Status,
		ExpiresAt: db.TimeNS(u.ExpiresAt), CreatedAt: db.TimeNS(u.Now), UpdatedAt: db.TimeNS(u.Now),
	})
}

// ByID returns nil for an unknown id.
func (r Repo) ByID(ctx context.Context, id string) (*Upload, error) {
	row, err := db.Q(r.DB).GetUpload(ctx, id)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u := Upload{
		ID: row.ID, Bucket: row.Bucket, ObjectKey: row.ObjectKey, Disk: row.Disk,
		OriginalName: row.OriginalName, Status: row.Status,
	}
	if row.PartSize.Valid {
		u.PartSize = int(row.PartSize.Int64)
	}
	if row.UserID.Valid {
		u.UserID = &row.UserID.Int64
	}
	if row.Path.Valid {
		u.Path = &row.Path.String
	}
	if row.MimeType.Valid {
		u.MimeType = &row.MimeType.String
	}
	if row.Size.Valid {
		u.Size = &row.Size.Int64
	}
	if row.TotalParts.Valid {
		n := int(row.TotalParts.Int64)
		u.TotalParts = &n
	}
	if tm, ok := db.ParseTime(row.ExpiresAt); ok {
		u.ExpiresAt = &tm
	}
	return &u, nil
}

// Complete records the assembled file.
func (r Repo) Complete(ctx context.Context, id, path, status string, parts int, now time.Time) error {
	return db.Q(r.DB).CompleteUpload(ctx, sqlc.CompleteUploadParams{
		Path: db.NS(path), TotalParts: db.NI(int64(parts)), Status: status,
		CompletedAt: db.TimeNS(now), UpdatedAt: db.TimeNS(now), ID: id,
	})
}

func (r Repo) SetStatus(ctx context.Context, id, status string, now time.Time) error {
	return db.Q(r.DB).SetUploadStatus(ctx, sqlc.SetUploadStatusParams{Status: status, UpdatedAt: db.TimeNS(now), ID: id})
}

// ExpiredPending lists the ids of pending uploads whose time ran out by now.
func (r Repo) ExpiredPending(ctx context.Context, now time.Time) ([]string, error) {
	return db.Q(r.DB).ListExpiredPendingUploads(ctx, db.TimeNS(now))
}

// Usage is how many uploads of a bucket are in a status, and their bytes.
type Usage struct {
	Bucket, Status string
	Count          int
	Bytes          int64
}

// UsageByBucket sums the uploads per bucket and status.
func (r Repo) UsageByBucket(ctx context.Context) ([]Usage, error) {
	rows, err := db.Q(r.DB).UploadUsageByBucket(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Usage, len(rows))
	for i, row := range rows {
		out[i] = Usage{Bucket: row.Bucket, Status: row.Status, Count: int(row.Count), Bytes: sumBytes(row.Coalesce)}
	}
	return out, nil
}

// sumBytes reads COALESCE(SUM(size), 0), which the driver types loosely.
func sumBytes(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	}
	return 0
}
