package uploads

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type uploadBucketCfg struct {
	Prefix   string
	MaxSize  int64
	Exts     []string
	Mimes    []string
	PartSize int
}

func uploadBucket(name string) (uploadBucketCfg, bool) {
	if name != "transaction-imports" {
		return uploadBucketCfg{}, false
	}
	return uploadBucketCfg{
		Prefix: "transaction-imports", MaxSize: 512 * 1024 * 1024, PartSize: defaultPartSize,
		Exts:  []string{"csv", "txt"},
		Mimes: []string{"text/csv", "text/plain", "application/csv", "application/vnd.ms-excel", "application/octet-stream"},
	}, true
}

func guardUpload(b uploadBucketCfg, filename string, size *int64, mime *string) error {
	if size != nil {
		if *size < 1 {
			return fmt.Errorf("empty")
		}
		if b.MaxSize > 0 && *size > b.MaxSize {
			return fmt.Errorf("too large")
		}
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	if len(b.Exts) > 0 {
		ok := false
		for _, e := range b.Exts {
			if e == ext {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("type not allowed")
		}
	}
	if mime != nil && *mime != "" && len(b.Mimes) > 0 {
		ok := false
		for _, m := range b.Mimes {
			if m == *mime {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("content type")
		}
	}
	return nil
}

func buildObjectKey(prefix, id, filename string) string {
	date := time.Now().UTC().Format("2006/01/02")
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	slug := url.PathEscape(strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		if r == ' ' {
			return '-'
		}
		return -1
	}, strings.ToLower(base)))
	if slug == "" {
		slug = "file"
	}
	name := slug
	if ext != "" {
		name += "." + ext
	}
	return strings.Trim(prefix, "/") + "/" + date + "/" + id + "/" + name
}

func newUploadID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func partETag(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := md5.New()
	_, _ = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil))
}
