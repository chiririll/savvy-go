package monitoring

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/importer"
	"savvy-go/internal/store"
	"savvy-go/internal/uploads"
	"savvy-go/internal/version"
)

var processStarted = time.Now()

// Handler is the monitoring of the instance: disk, uploads, imports, process.
type Handler struct {
	Store      store.Store
	Uploads    uploads.Service
	UploadsDir string
}

func (h Handler) storage(w http.ResponseWriter, r *http.Request) {
	managed, uploads := h.uploadStats(r.Context())
	importBy, importTotal := h.importStats(r.Context())
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"volume":  volumeStats(h.UploadsDir),
		"managed": managed,
		"uploads": uploads,
		"imports": map[string]any{"total": importTotal, "by_status": importBy},
	})
}

// volumeStats describes the disk the uploads live on; sizes are null when unknown.
func volumeStats(root string) map[string]any {
	total, free, ok := VolumeSpace(root)
	var used, usedPct, totalB, freeB any
	if ok {
		u := max(total-free, 0)
		used, totalB, freeB = u, total, free
		if total > 0 {
			usedPct = float64(int(float64(u)/float64(total)*1000+0.5)) / 10
		}
	}
	return map[string]any{
		"disk": "local", "path": root,
		"total_bytes": totalB, "free_bytes": freeB, "used_bytes": used, "used_percent": usedPct,
	}
}

// uploadStats sums the managed uploads per bucket and counts uploads per status.
func (h Handler) uploadStats(ctx context.Context) (managed, uploads map[string]any) {
	usage, _ := h.Uploads.UsageByBucket(ctx)
	type bucketAgg struct {
		bytes   int64
		objects int
	}
	byBucket := map[string]*bucketAgg{}
	byStatus := map[string]int{}
	var managedUsed, pendingBytes int64
	var objects int
	for _, u := range usage {
		byStatus[u.Status] += u.Count
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
	buckets := []map[string]any{}
	for name, agg := range byBucket {
		buckets = append(buckets, map[string]any{"bucket": name, "bytes": agg.bytes, "objects": agg.objects})
	}
	total := 0
	for _, n := range byStatus {
		total += n
	}
	managed = map[string]any{"used_bytes": managedUsed, "objects": objects, "pending_bytes": pendingBytes, "buckets": buckets}
	return managed, map[string]any{"total": total, "by_status": byStatus}
}

// importStats counts imports by status across all spaces.
func (h Handler) importStats(ctx context.Context) (byStatus map[string]int, total int) {
	byStatus = map[string]int{}
	ids, _ := h.Store.Spaces(ctx)
	for _, id := range ids {
		d, err := h.Store.Space(ctx, id)
		if err != nil {
			continue
		}
		counts, err := importer.Service{DB: d}.CountsByStatus(ctx)
		if err != nil {
			continue
		}
		for status, n := range counts {
			byStatus[status] += n
			total += n
		}
	}
	return byStatus, total
}

func (Handler) resources(w http.ResponseWriter, r *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"cpu":    cpuSnapshot(),
		"memory": memorySnapshot(),
		"process": map[string]any{
			"memory_bytes": ms.Alloc, "peak_bytes": ms.Sys, "limit_bytes": nil,
			"goroutines": runtime.NumGoroutine(), "gc_cycles": ms.NumGC,
		},
		"runtime": map[string]any{
			"go_version": runtime.Version(), "app_version": version.Value,
			"environment": version.Env, "uptime_seconds": int(time.Since(processStarted).Seconds()),
		},
	})
}

// VolumeSpace is the size and free space of the volume holding path, or of its
// nearest existing parent.
func VolumeSpace(path string) (total, free int64, ok bool) {
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
