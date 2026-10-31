package sysinfo

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Linux reads /proc and the cgroup filesystem under Root ("/" on a real host).
// NumCPU is the host's cores, runtime.NumCPU when zero.
type Linux struct {
	Root   string
	NumCPU int
}

// unlimited is the size above which a cgroup limit means "no limit": v1 reports
// a huge page-aligned number instead of "max".
const unlimited = int64(1) << 60

func (l Linux) path(p ...string) string { return filepath.Join(append([]string{l.Root}, p...)...) }

func (l Linux) CPU() (CPU, bool) {
	b, err := os.ReadFile(l.path("proc", "loadavg"))
	if err != nil {
		return CPU{}, false
	}
	f := strings.Fields(string(b))
	var c CPU
	if len(f) < 3 {
		return CPU{}, false
	}
	for i := range c.Load {
		v, err := strconv.ParseFloat(f[i], 64)
		if err != nil || v < 0 {
			return CPU{}, false
		}
		c.Load[i] = v
	}
	c.Cores = l.cores()
	return c, true
}

// cores is what the process may use: the host's cores, or the container's CPU
// quota when that is smaller.
func (l Linux) cores() float64 {
	n := float64(l.NumCPU)
	if n <= 0 {
		n = float64(runtime.NumCPU())
	}
	if quota, ok := l.cpuQuota(); ok {
		n = min(n, quota)
	}
	return n
}

// cpuQuota is the container's CPU limit in cores (cgroup v2, else v1); ok is
// false when there is none.
func (l Linux) cpuQuota() (cores float64, ok bool) {
	if b, err := os.ReadFile(l.path("sys", "fs", "cgroup", "cpu.max")); err == nil {
		f := strings.Fields(string(b))
		if len(f) != 2 || f[0] == "max" {
			return 0, false
		}
		return quotaCores(f[0], f[1])
	}
	quota, ok1 := l.readInt("sys", "fs", "cgroup", "cpu", "cpu.cfs_quota_us")
	period, ok2 := l.readInt("sys", "fs", "cgroup", "cpu", "cpu.cfs_period_us")
	if !ok1 || !ok2 {
		return 0, false // v1 writes -1 for no quota, which readInt rejects
	}
	return quotaCores(strconv.FormatInt(quota, 10), strconv.FormatInt(period, 10))
}

func quotaCores(quota, period string) (float64, bool) {
	q, err1 := strconv.ParseFloat(quota, 64)
	p, err2 := strconv.ParseFloat(period, 64)
	if err1 != nil || err2 != nil || q <= 0 || p <= 0 {
		return 0, false
	}
	return q / p, true
}

func (l Linux) Memory() (Memory, bool) {
	host, hostOK := l.hostMemory()
	if lim, used, ok := l.cgroupMemory(); ok && (!hostOK || lim < host.Total) {
		used = min(max(used, 0), lim)
		return Memory{Total: lim, Used: used, Free: lim - used, Source: "cgroup"}, true
	}
	return host, hostOK
}

// hostMemory uses MemAvailable, not MemFree, so reclaimable cache is not "used".
func (l Linux) hostMemory() (Memory, bool) {
	f, err := os.Open(l.path("proc", "meminfo"))
	if err != nil {
		return Memory{}, false
	}
	defer f.Close()
	var total, avail int64
	var haveTotal, haveAvail bool
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		name, rest, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		switch name {
		case "MemTotal":
			total, haveTotal = parseKB(rest)
		case "MemAvailable":
			avail, haveAvail = parseKB(rest)
		}
	}
	if !haveTotal || !haveAvail {
		return Memory{}, false
	}
	return physicalMemory(uint64(total), uint64(avail))
}

func parseKB(s string) (int64, bool) {
	f := strings.Fields(s)
	if len(f) == 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(f[0], 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n * 1024, true
}

// cgroupMemory is the container limit and working set: cgroup v2, else v1.
// ok is false when there is no limit.
func (l Linux) cgroupMemory() (limit, used int64, ok bool) {
	if lim, ok := l.readInt("sys", "fs", "cgroup", "memory.max"); ok && lim < unlimited {
		cur, ok := l.readInt("sys", "fs", "cgroup", "memory.current")
		if !ok {
			return 0, 0, false
		}
		return lim, cur - l.statValue(l.path("sys", "fs", "cgroup", "memory.stat"), "inactive_file"), true
	}
	if lim, ok := l.readInt("sys", "fs", "cgroup", "memory", "memory.limit_in_bytes"); ok && lim < unlimited {
		cur, ok := l.readInt("sys", "fs", "cgroup", "memory", "memory.usage_in_bytes")
		if !ok {
			return 0, 0, false
		}
		return lim, cur - l.statValue(l.path("sys", "fs", "cgroup", "memory", "memory.stat"), "total_inactive_file"), true
	}
	return 0, 0, false
}

// readInt parses a one-number file. "max" and v1's huge sentinel are not a limit.
func (l Linux) readInt(p ...string) (int64, bool) {
	b, err := os.ReadFile(l.path(p...))
	if err != nil {
		return 0, false
	}
	s := strings.TrimSpace(string(b))
	if s == "max" {
		return unlimited, true
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// statValue is one counter of a "key value" file; 0 when absent.
func (Linux) statValue(file, key string) int64 {
	b, err := os.ReadFile(file)
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(line, " ")
		if ok && k == key {
			n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			return n
		}
	}
	return 0
}
