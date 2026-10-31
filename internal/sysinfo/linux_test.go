package sysinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func fakeRoot(t *testing.T, files map[string]string) Linux {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Linux{Root: root}
}

const meminfo = "MemTotal:        4096000 kB\nMemFree:          100000 kB\nMemAvailable:    1024000 kB\n"

func TestLinuxCPU(t *testing.T) {
	c, ok := fakeRoot(t, map[string]string{"proc/loadavg": "0.42 0.61 0.55 1/234 5678\n"}).CPU()
	if !ok || c.Load != [3]float64{0.42, 0.61, 0.55} {
		t.Fatalf("got %v %v", c, ok)
	}
}

func TestLinuxCPUUnreadable(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"missing":   {},
		"garbage":   {"proc/loadavg": "x y z"},
		"truncated": {"proc/loadavg": "0.1 0.2"},
	} {
		if _, ok := fakeRoot(t, files).CPU(); ok {
			t.Errorf("%s: want not ok", name)
		}
	}
}

func TestLinuxMemoryHost(t *testing.T) {
	m, ok := fakeRoot(t, map[string]string{"proc/meminfo": meminfo}).Memory()
	want := Memory{Total: 4096000 * 1024, Used: 3072000 * 1024, Free: 1024000 * 1024, Source: "host"}
	if !ok || m != want {
		t.Fatalf("got %+v %v", m, ok)
	}
}

func TestLinuxMemoryCgroupV2(t *testing.T) {
	m, ok := fakeRoot(t, map[string]string{
		"proc/meminfo":                 meminfo,
		"sys/fs/cgroup/memory.max":     "2147483648\n",
		"sys/fs/cgroup/memory.current": "1073741824\n",
		"sys/fs/cgroup/memory.stat":    "anon 1\ninactive_file 268435456\n",
	}).Memory()
	want := Memory{Total: 2 << 30, Used: 1<<30 - 1<<28, Free: 2<<30 - (1<<30 - 1<<28), Source: "cgroup"}
	if !ok || m != want {
		t.Fatalf("got %+v %v", m, ok)
	}
}

func TestLinuxMemoryCgroupV1(t *testing.T) {
	m, ok := fakeRoot(t, map[string]string{
		"proc/meminfo": meminfo,
		"sys/fs/cgroup/memory/memory.limit_in_bytes": "1073741824",
		"sys/fs/cgroup/memory/memory.usage_in_bytes": "536870912",
	}).Memory()
	if !ok || m.Source != "cgroup" || m.Total != 1<<30 || m.Used != 1<<29 {
		t.Fatalf("got %+v %v", m, ok)
	}
}

func TestLinuxMemoryNoCgroupLimit(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"v2 max": {"sys/fs/cgroup/memory.max": "max", "sys/fs/cgroup/memory.current": "5"},
		"v1 sentinel": {
			"sys/fs/cgroup/memory/memory.limit_in_bytes": "9223372036854771712",
			"sys/fs/cgroup/memory/memory.usage_in_bytes": "5",
		},
		"limit above host": {"sys/fs/cgroup/memory.max": "99999999999999", "sys/fs/cgroup/memory.current": "5"},
	} {
		files["proc/meminfo"] = meminfo
		if m, ok := fakeRoot(t, files).Memory(); !ok || m.Source != "host" {
			t.Errorf("%s: got %+v %v", name, m, ok)
		}
	}
}

func TestLinuxMemoryUnreadable(t *testing.T) {
	if _, ok := fakeRoot(t, nil).Memory(); ok {
		t.Fatal("want not ok")
	}
	if _, ok := fakeRoot(t, map[string]string{"proc/meminfo": "MemTotal: 5 kB\n"}).Memory(); ok {
		t.Fatal("no MemAvailable: want not ok")
	}
}

func TestLinuxCores(t *testing.T) {
	load := "proc/loadavg"
	const avg = "1 1 1 1/1 1"
	for name, tc := range map[string]struct {
		files map[string]string
		want  float64
	}{
		"no quota":       {map[string]string{load: avg}, 8},
		"v2 max":         {map[string]string{load: avg, "sys/fs/cgroup/cpu.max": "max 100000"}, 8},
		"v2 quota":       {map[string]string{load: avg, "sys/fs/cgroup/cpu.max": "150000 100000"}, 1.5},
		"v2 above host":  {map[string]string{load: avg, "sys/fs/cgroup/cpu.max": "2000000 100000"}, 8},
		"v1 quota":       {map[string]string{load: avg, "sys/fs/cgroup/cpu/cpu.cfs_quota_us": "50000", "sys/fs/cgroup/cpu/cpu.cfs_period_us": "100000"}, 0.5},
		"v1 unlimited":   {map[string]string{load: avg, "sys/fs/cgroup/cpu/cpu.cfs_quota_us": "-1", "sys/fs/cgroup/cpu/cpu.cfs_period_us": "100000"}, 8},
		"v2 garbage":     {map[string]string{load: avg, "sys/fs/cgroup/cpu.max": "x y"}, 8},
		"v2 zero period": {map[string]string{load: avg, "sys/fs/cgroup/cpu.max": "100 0"}, 8},
	} {
		l := fakeRoot(t, tc.files)
		l.NumCPU = 8
		if c, ok := l.CPU(); !ok || c.Cores != tc.want {
			t.Errorf("%s: cores = %v (ok %v), want %v", name, c.Cores, ok, tc.want)
		}
	}
}
