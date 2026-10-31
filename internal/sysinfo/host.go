package sysinfo

import (
	"math"
	"runtime"
)

// CPU is the host load average over 1, 5 and 15 minutes, and the cores it is
// shared by: fewer than the host has when a container is limited to a quota.
type CPU struct {
	Load  [3]float64
	Cores float64
}

// Memory is what the server may use. Source is "cgroup" when a container limit
// applies, "host" otherwise.
type Memory struct {
	Total, Used, Free int64
	Source            string
}

// Collector reads host figures. A figure that cannot be read reports ok false,
// never zero.
type Collector interface {
	CPU() (cpu CPU, ok bool)
	Memory() (mem Memory, ok bool)
}

// Unsupported reads nothing: the platform has no collector.
type Unsupported struct{}

func (Unsupported) CPU() (CPU, bool)       { return CPU{}, false }
func (Unsupported) Memory() (Memory, bool) { return Memory{}, false }

// Default is the collector for this host: Linux reads /proc and the cgroups,
// Windows and the BSDs (macOS included) what their own APIs give, and anything
// else knows nothing.
func Default() Collector {
	if runtime.GOOS == "linux" {
		return Linux{Root: "/"}
	}
	return native()
}

// LoadPercent is the 1-minute load as a share of the cores, to 1 decimal. It can
// pass 100 when the host is overloaded.
func (c CPU) LoadPercent() (pct float64, ok bool) {
	if c.Cores <= 0 {
		return 0, false
	}
	return round1(c.Load[0] / c.Cores * 100), true
}

// UsedPercent is the used share of the memory, to 1 decimal.
func (m Memory) UsedPercent() (pct float64, ok bool) {
	if m.Total <= 0 {
		return 0, false
	}
	return round1(float64(m.Used) / float64(m.Total) * 100), true
}

func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }

// physicalMemory is the memory of a host that reports total and available.
func physicalMemory(total, avail uint64) (Memory, bool) {
	if total == 0 || total > math.MaxInt64 {
		return Memory{}, false
	}
	t, a := int64(total), int64(min(avail, total))
	return Memory{Total: t, Used: t - a, Free: a, Source: "host"}, true
}
