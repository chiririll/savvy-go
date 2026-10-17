package uploads

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	repo "savvy-go/internal/db/repo/uploads"
	"savvy-go/internal/store"
)

const (
	uploadPending   = "pending"
	uploadCompleted = "completed"
	uploadAborted   = "aborted"
	uploadConsumed  = "consumed"
	defaultPartSize = 8 * 1024 * 1024
	maxParts        = 10000
	uploadURLTTL    = 3600
)

// Completed reports whether every part has arrived and the file is assembled.
func (u Upload) Completed() bool { return u.Status == uploadCompleted }

type Upload struct {
	ID           string
	UserID       *int64
	Bucket       string
	ObjectKey    string
	Disk         string
	Path         *string
	OriginalName string
	MimeType     *string
	Size         *int64
	PartSize     int
	TotalParts   *int
	Status       string
	ExpiresAt    *time.Time
}

type Service struct {
	DB         store.DB
	Root       string
	SignSecret string
}

func (s Service) repo() repo.Repo { return repo.Repo{DB: s.DB} }

// Usage is how many uploads of a bucket are in a status, and their bytes.
type Usage = repo.Usage

// UsageByBucket sums the uploads per bucket and status.
func (s Service) UsageByBucket(ctx context.Context) ([]Usage, error) {
	return s.repo().UsageByBucket(ctx)
}

func (s Service) Create(
	ctx context.Context,
	userID int64,
	bucket, filename string,
	size *int64,
	mime *string,
) (*Upload, error) {
	b, ok := uploadBucket(bucket)
	if !ok {
		return nil, fmt.Errorf("unknown bucket")
	}
	if err := guardUpload(b, filename, size, mime); err != nil {
		return nil, err
	}
	partSize := b.PartSize
	if partSize <= 0 {
		partSize = defaultPartSize
	}
	if size != nil {
		needed := int((*size + int64(maxParts) - 1) / maxParts)
		if needed > partSize {
			mib := 1024 * 1024
			partSize = ((needed + mib - 1) / mib) * mib
		}
	}
	id := newUploadID()
	key := buildObjectKey(b.Prefix, id, filename)
	now := time.Now().UTC()
	exp := now.Add(time.Duration(uploadURLTTL) * time.Second)
	var totalParts *int
	if size != nil {
		n := int((*size + int64(partSize) - 1) / int64(partSize))
		if n < 1 {
			n = 1
		}
		totalParts = &n
	}
	err := s.repo().Insert(ctx, repo.New{
		ID: id, UserID: userID, Bucket: bucket, ObjectKey: key, Disk: "local", OriginalName: filename,
		MimeType: mime, Size: size, PartSize: partSize, TotalParts: totalParts, Status: uploadPending,
		ExpiresAt: exp, Now: now,
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func (s Service) ByID(ctx context.Context, id string) (*Upload, error) {
	row, err := s.repo().ByID(ctx, id)
	if row == nil || err != nil {
		return nil, err
	}
	return &Upload{
		ID: row.ID, UserID: row.UserID, Bucket: row.Bucket, ObjectKey: row.ObjectKey, Disk: row.Disk,
		Path: row.Path, OriginalName: row.OriginalName, MimeType: row.MimeType, Size: row.Size,
		PartSize: row.PartSize, TotalParts: row.TotalParts, Status: row.Status, ExpiresAt: row.ExpiresAt,
	}, nil
}

func (s Service) SignPart(u *Upload, part int) (map[string]any, error) {
	if u.Status != uploadPending {
		return nil, fmt.Errorf("not accepting")
	}
	if part < 1 || part > maxParts {
		return nil, fmt.Errorf("part out of range")
	}
	expires := time.Now().UTC().Add(time.Duration(uploadURLTTL) * time.Second).Unix()
	sig := s.sign(u.ID, part, expires)
	// Relative on purpose: the browser resolves it against the origin it loaded the app from,
	// so uploads work behind any port or proxy regardless of APP_URL.
	raw := fmt.Sprintf("/api/uploads/%s/parts/%d?expires=%d&signature=%s", u.ID, part, expires, sig)
	return map[string]any{
		"url": raw, "method": "PUT", "headers": map[string]string{}, "expires": uploadURLTTL,
	}, nil
}

func (s Service) VerifyPart(id string, part int, expires int64, signature string) bool {
	if expires < time.Now().UTC().Unix() {
		return false
	}
	want := s.sign(id, part, expires)
	return hmac.Equal([]byte(want), []byte(signature))
}

func (s Service) sign(id string, part int, expires int64) string {
	mac := hmac.New(sha256.New, []byte(s.SignSecret))
	_, _ = fmt.Fprintf(mac, "%s|%d|%d", id, part, expires)
	return hex.EncodeToString(mac.Sum(nil))
}

func (s Service) StorePart(u *Upload, part int, body io.Reader) (string, int64, error) {
	if u.Status != uploadPending {
		return "", 0, fmt.Errorf("not accepting")
	}
	dir := filepath.Join(s.Root, u.ID, "parts")
	if err := os.MkdirAll(dir, 0o775); err != nil {
		return "", 0, err
	}
	path := filepath.Join(dir, strconv.Itoa(part))
	f, err := os.Create(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := md5.New()
	n, err := io.Copy(io.MultiWriter(f, h), body)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func (s Service) ListParts(u *Upload) []map[string]any {
	dir := filepath.Join(s.Root, u.ID, "parts")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []map[string]any{}
	}
	var nums []int
	for _, e := range entries {
		n, err := strconv.Atoi(e.Name())
		if err == nil {
			nums = append(nums, n)
		}
	}
	sort.Ints(nums)
	var out []map[string]any
	for _, n := range nums {
		info, err := os.Stat(filepath.Join(dir, strconv.Itoa(n)))
		if err != nil {
			continue
		}
		etag := partETag(filepath.Join(dir, strconv.Itoa(n)))
		out = append(out, map[string]any{"PartNumber": n, "Size": info.Size(), "ETag": etag})
	}
	return out
}

func (s Service) Complete(ctx context.Context, u *Upload, parts []struct {
	Number int
	ETag   string
}) (*Upload, error) {
	if u.Status == uploadCompleted {
		return u, nil
	}
	if u.Status != uploadPending {
		return nil, fmt.Errorf("cannot complete")
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("part required")
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].Number < parts[j].Number })
	dest := filepath.Join(s.Root, filepath.FromSlash(u.ObjectKey))
	if err := os.MkdirAll(filepath.Dir(dest), 0o775); err != nil {
		return nil, err
	}
	out, err := os.Create(dest)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	for _, p := range parts {
		src, err := os.Open(filepath.Join(s.Root, u.ID, "parts", strconv.Itoa(p.Number)))
		if err != nil {
			return nil, fmt.Errorf("missing part %d", p.Number)
		}
		_, err = io.Copy(out, src)
		src.Close()
		if err != nil {
			return nil, err
		}
	}
	err = s.repo().Complete(ctx, u.ID, u.ObjectKey, uploadCompleted, len(parts), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	_ = os.RemoveAll(filepath.Join(s.Root, u.ID, "parts"))
	return s.ByID(ctx, u.ID)
}

func (s Service) Abort(ctx context.Context, u *Upload) error {
	_ = os.RemoveAll(filepath.Join(s.Root, u.ID))
	if u.Path != nil {
		_ = os.Remove(filepath.Join(s.Root, filepath.FromSlash(*u.Path)))
	}
	return s.repo().SetStatus(ctx, u.ID, uploadAborted, time.Now().UTC())
}

func (s Service) ReadFile(u *Upload) ([]byte, error) {
	if u.Status != uploadCompleted || u.Path == nil {
		return nil, fmt.Errorf("not ready")
	}
	return os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(*u.Path)))
}

func (s Service) Discard(ctx context.Context, u *Upload) error {
	if u.Path != nil {
		_ = os.Remove(filepath.Join(s.Root, filepath.FromSlash(*u.Path)))
	}
	_ = os.RemoveAll(filepath.Join(s.Root, u.ID))
	return s.repo().SetStatus(ctx, u.ID, uploadConsumed, time.Now().UTC())
}

func (s Service) Location(u *Upload) string {
	return "/api/s3/" + u.Bucket + "/" + strings.TrimLeft(u.ObjectKey, "/")
}

func (s Service) PruneExpired(ctx context.Context) error {
	ids, err := s.repo().ExpiredPending(ctx, time.Now().UTC())
	if err != nil {
		return err
	}
	for _, id := range ids {
		if u, _ := s.ByID(ctx, id); u != nil {
			_ = s.Abort(ctx, u)
		}
	}
	return nil
}
