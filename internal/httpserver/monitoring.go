package httpserver

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chiririll/savvy-plus/internal/version"
)

var processStarted = time.Now()

func (s *Server) monitoringStorage(w http.ResponseWriter, r *http.Request) {
	root := s.cfg.UploadsDir
	total, free, ok := volumeSpace(root)
	var used any
	var usedPct any
	var totalB, freeB any
	if ok {
		u := total - free
		if u < 0 {
			u = 0
		}
		used = u
		totalB = total
		freeB = free
		if total > 0 {
			usedPct = float64(int(float64(u)/float64(total)*1000+0.5)) / 10
		}
	}

	type bucketRow struct {
		Bucket string
		Status string
		Count  int
		Bytes  int64
	}
	rows, err := s.db.QueryContext(r.Context(), `
		SELECT bucket, status, COUNT(*), COALESCE(SUM(size), 0)
		FROM uploads GROUP BY bucket, status`)
	var usage []bucketRow
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var b bucketRow
			if rows.Scan(&b.Bucket, &b.Status, &b.Count, &b.Bytes) == nil {
				usage = append(usage, b)
			}
		}
	}

	type bucketAgg struct {
		bytes   int64
		objects int
	}
	byBucket := map[string]*bucketAgg{}
	uploadsByStatus := map[string]int{}
	var managedUsed, pendingBytes int64
	var objects int
	for _, u := range usage {
		uploadsByStatus[u.Status] += u.Count
		if u.Status == "pending" || u.Status == "completed" {
			managedUsed += u.Bytes
			if byBucket[u.Bucket] == nil {
				byBucket[u.Bucket] = &bucketAgg{}
			}
			byBucket[u.Bucket].bytes += u.Bytes
			if u.Status == "completed" {
				byBucket[u.Bucket].objects += u.Count
				objects += u.Count
			}
		}
		if u.Status == "pending" {
			pendingBytes += u.Bytes
		}
	}
	var buckets []map[string]any
	for name, agg := range byBucket {
		buckets = append(buckets, map[string]any{"bucket": name, "bytes": agg.bytes, "objects": agg.objects})
	}
	if buckets == nil {
		buckets = []map[string]any{}
	}
	uploadTotal := 0
	for _, n := range uploadsByStatus {
		uploadTotal += n
	}

	importBy := map[string]int{}
	importTotal := 0
	if irows, err := s.db.QueryContext(r.Context(), `SELECT status, COUNT(*) FROM transaction_imports GROUP BY status`); err == nil {
		defer irows.Close()
		for irows.Next() {
			var st string
			var n int
			if irows.Scan(&st, &n) == nil {
				importBy[st] = n
				importTotal += n
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"volume": map[string]any{
			"disk": "local", "path": root,
			"total_bytes": totalB, "free_bytes": freeB, "used_bytes": used, "used_percent": usedPct,
		},
		"managed": map[string]any{
			"used_bytes": managedUsed, "objects": objects, "pending_bytes": pendingBytes, "buckets": buckets,
		},
		"uploads": map[string]any{"total": uploadTotal, "by_status": uploadsByStatus},
		"imports": map[string]any{"total": importTotal, "by_status": importBy},
	})
}

func (s *Server) monitoringResources(w http.ResponseWriter, r *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	writeJSON(w, http.StatusOK, map[string]any{
		"cpu":     cpuSnapshot(),
		"memory":  memorySnapshot(),
		"process": map[string]any{"memory_bytes": ms.Alloc, "peak_bytes": ms.Sys, "limit_bytes": nil},
		"queue":   map[string]any{"pending": nil, "reserved": nil, "failed": nil},
		"runtime": map[string]any{
			"php_version": runtime.Version(), "laravel_version": version.Value,
			"environment": version.Env, "uptime_seconds": int(time.Since(processStarted).Seconds()),
		},
	})
}

func volumeSpace(path string) (total, free int64, ok bool) {
	probe := path
	for probe != "" {
		if st, err := os.Stat(probe); err == nil && st.IsDir() {
			return diskUsage(probe)
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		probe = parent
	}
	return 0, 0, false
}

func cpuSnapshot() map[string]any {
	cores := float64(runtime.NumCPU())
	return map[string]any{"cores": cores, "load": nil, "load_percent": nil}
}

func memorySnapshot() map[string]any {
	return map[string]any{
		"total_bytes": nil, "used_bytes": nil, "free_bytes": nil, "used_percent": nil, "source": nil,
	}
}
