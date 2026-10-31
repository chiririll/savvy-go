//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package sysinfo

import (
	"encoding/binary"
	"runtime"

	"golang.org/x/sys/unix"
)

// bsd knows the load average only: a usable "used memory" figure needs
// per-system calls (Mach on macOS) that this package does without.
type bsd struct{}

func native() Collector { return bsd{} }

func (bsd) Memory() (Memory, bool) { return Memory{}, false }

func (bsd) CPU() (CPU, bool) {
	raw, err := unix.SysctlRaw("vm.loadavg")
	if err != nil {
		return CPU{}, false
	}
	load, ok := parseLoadavg(raw, binary.NativeEndian)
	if !ok {
		return CPU{}, false
	}
	return CPU{Load: load, Cores: float64(runtime.NumCPU())}, true
}
