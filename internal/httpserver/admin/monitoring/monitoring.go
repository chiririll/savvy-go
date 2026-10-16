package monitoring

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/importer"
	"savvy-go/internal/platform/sysinfo"
	"savvy-go/internal/store"
	"savvy-go/internal/uploads"
)

var processStarted = time.Now()

// Handler is the monitoring of the instance: disk, uploads, imports, process.
type Handler struct {
	Store      store.Store
	Uploads    uploads.Service
	UploadsDir string
	Version    string // the release
	Env        string // the build environment
}

func (h Handler) storage(w http.ResponseWriter, r *http.Request) {
	managed, uploads := h.uploadStats(r.Context())
	importBy, importTotal := h.importStats(r.Context())
	httpx.WriteJSON(w, http.StatusOK, storageBody{
		Volume:  volumeStats(h.UploadsDir),
		Managed: managed,
		Uploads: uploads,
		Imports: statusCounts{Total: importTotal, ByStatus: importBy},
	})
}

// volumeStats describes the disk the uploads live on; sizes are null when unknown.
func volumeStats(root string) volumeBody {
	out := volumeBody{Disk: "local", Path: root}
	if total, free, ok := sysinfo.VolumeSpace(root); ok {
		used := max(total-free, 0)
		out.TotalBytes, out.FreeBytes, out.UsedBytes = &total, &free, &used
		if total > 0 {
			pct := float64(int(float64(used)/float64(total)*1000+0.5)) / 10
			out.UsedPercent = &pct
		}
	}
	return out
}

// storageBody, volumeBody, managedBody, bucketBody and statusCounts are the
// storage page: the disk, what the uploads take of it, and counts by status.
type storageBody struct {
	Volume  volumeBody   `json:"volume"`
	Managed managedBody  `json:"managed"`
	Uploads statusCounts `json:"uploads"`
	Imports statusCounts `json:"imports"`
}

// volumeBody has null sizes when the disk cannot be measured.
type volumeBody struct {
	Disk        string   `json:"disk"`
	Path        string   `json:"path"`
	TotalBytes  *int64   `json:"total_bytes"`
	FreeBytes   *int64   `json:"free_bytes"`
	UsedBytes   *int64   `json:"used_bytes"`
	UsedPercent *float64 `json:"used_percent"`
}

type managedBody struct {
	UsedBytes    int64        `json:"used_bytes"`
	Objects      int          `json:"objects"`
	PendingBytes int64        `json:"pending_bytes"`
	Buckets      []bucketBody `json:"buckets"`
}

type bucketBody struct {
	Bucket  string `json:"bucket"`
	Bytes   int64  `json:"bytes"`
	Objects int    `json:"objects"`
}

type statusCounts struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
}

// uploadStats sums the managed uploads per bucket and counts uploads per status.
func (h Handler) uploadStats(ctx context.Context) (managed managedBody, uploads statusCounts) {
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
	buckets := []bucketBody{}
	for name, agg := range byBucket {
		buckets = append(buckets, bucketBody{Bucket: name, Bytes: agg.bytes, Objects: agg.objects})
	}
	total := 0
	for _, n := range byStatus {
		total += n
	}
	managed = managedBody{UsedBytes: managedUsed, Objects: objects, PendingBytes: pendingBytes, Buckets: buckets}
	return managed, statusCounts{Total: total, ByStatus: byStatus}
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

func (h Handler) resources(w http.ResponseWriter, r *http.Request) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	httpx.WriteJSON(w, http.StatusOK, resourcesBody{
		CPU:    cpuBody{Cores: float64(runtime.NumCPU())},
		Memory: memoryBody{},
		Process: processBody{
			MemoryBytes: ms.Alloc, PeakBytes: ms.Sys, Goroutines: runtime.NumGoroutine(), GCCycles: ms.NumGC,
		},
		Runtime: runtimeBody{
			GoVersion: runtime.Version(), AppVersion: h.Version,
			Environment: h.Env, UptimeSeconds: int(time.Since(processStarted).Seconds()),
		},
	})
}

// resourcesBody is the process page. The load and host memory figures are
// null: this process cannot measure them portably.
type resourcesBody struct {
	CPU     cpuBody     `json:"cpu"`
	Memory  memoryBody  `json:"memory"`
	Process processBody `json:"process"`
	Runtime runtimeBody `json:"runtime"`
}

type cpuBody struct {
	Cores       float64  `json:"cores"`
	Load        *float64 `json:"load"`
	LoadPercent *float64 `json:"load_percent"`
}

type memoryBody struct {
	TotalBytes  *int64   `json:"total_bytes"`
	UsedBytes   *int64   `json:"used_bytes"`
	FreeBytes   *int64   `json:"free_bytes"`
	UsedPercent *float64 `json:"used_percent"`
	Source      *string  `json:"source"`
}

type processBody struct {
	MemoryBytes uint64  `json:"memory_bytes"`
	PeakBytes   uint64  `json:"peak_bytes"`
	LimitBytes  *uint64 `json:"limit_bytes"`
	Goroutines  int     `json:"goroutines"`
	GCCycles    uint32  `json:"gc_cycles"`
}

type runtimeBody struct {
	GoVersion     string `json:"go_version"`
	AppVersion    string `json:"app_version"`
	Environment   string `json:"environment"`
	UptimeSeconds int    `json:"uptime_seconds"`
}
