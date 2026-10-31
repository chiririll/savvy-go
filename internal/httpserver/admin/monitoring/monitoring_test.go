package monitoring

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime/debug"
	"testing"

	"savvy-go/internal/sysinfo"
)

type fakeHost struct {
	cpu   sysinfo.CPU
	cpuOK bool
	mem   sysinfo.Memory
	memOK bool
}

func (f fakeHost) CPU() (sysinfo.CPU, bool)       { return f.cpu, f.cpuOK }
func (f fakeHost) Memory() (sysinfo.Memory, bool) { return f.mem, f.memOK }

func TestCPUStats(t *testing.T) {
	got := cpuStats(fakeHost{cpu: sysinfo.CPU{Load: [3]float64{1.5, 3.5, 2.8}, Cores: 2}, cpuOK: true}, 8)
	if len(got.Load) != 3 || got.Load[0] != 1.5 || got.Cores != 2 || got.LoadPercent == nil || *got.LoadPercent != 75 {
		t.Fatalf("got %+v", got)
	}
}

func TestGoMemoryLimit(t *testing.T) {
	if goMemoryLimit() != nil {
		t.Skip("GOMEMLIMIT is set in the environment")
	}
	defer debug.SetMemoryLimit(debug.SetMemoryLimit(512 << 20))
	if got := goMemoryLimit(); got == nil || *got != 512<<20 {
		t.Fatalf("got %v", got)
	}
}

func TestCPUStatsUnknown(t *testing.T) {
	for _, h := range []sysinfo.Collector{nil, sysinfo.Unsupported{}} {
		got := cpuStats(h, 4)
		if got.Cores != 4 || got.Load != nil || got.LoadPercent != nil {
			t.Fatalf("got %+v", got)
		}
	}
}

func TestMemoryStats(t *testing.T) {
	got := memoryStats(fakeHost{mem: sysinfo.Memory{Total: 2000, Used: 1200, Free: 800, Source: "cgroup"}, memOK: true})
	if got.TotalBytes == nil || *got.TotalBytes != 2000 || *got.UsedPercent != 60 || *got.Source != "cgroup" {
		t.Fatalf("got %+v", got)
	}
}

func TestMemoryStatsUnknown(t *testing.T) {
	for _, h := range []sysinfo.Collector{nil, sysinfo.Unsupported{}} {
		if got := memoryStats(h); got.TotalBytes != nil || got.Source != nil {
			t.Fatalf("got %+v", got)
		}
	}
}

func resourcesJSON(t *testing.T, host sysinfo.Service) map[string]map[string]any {
	t.Helper()
	rec := httptest.NewRecorder()
	Handler{Host: host}.resources(rec, httptest.NewRequest(http.MethodGet, "/monitoring/resources", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	var body map[string]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestResourcesEndpointFillsHostFigures(t *testing.T) {
	body := resourcesJSON(t, sysinfo.Service{Collector: fakeHost{
		cpu: sysinfo.CPU{Load: [3]float64{1, 2, 3}}, cpuOK: true,
		mem: sysinfo.Memory{Total: 1000, Used: 250, Free: 750, Source: "host"}, memOK: true,
	}})
	if load, _ := body["cpu"]["load"].([]any); len(load) != 3 {
		t.Errorf("cpu.load = %v", body["cpu"]["load"])
	}
	if body["memory"]["source"] != "host" || body["memory"]["used_percent"] != 25.0 {
		t.Errorf("memory = %v", body["memory"])
	}
}

func TestResourcesEndpointNullWithoutCollector(t *testing.T) {
	body := resourcesJSON(t, sysinfo.Service{})
	if body["cpu"]["load"] != nil || body["cpu"]["load_percent"] != nil {
		t.Errorf("cpu = %v", body["cpu"])
	}
	if body["memory"]["total_bytes"] != nil || body["memory"]["source"] != nil {
		t.Errorf("memory = %v", body["memory"])
	}
}
