//go:build windows

package sysinfo

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows has no load average, so only memory is known.
type nativeWindows struct{}

func native() Collector { return nativeWindows{} }

func (nativeWindows) CPU() (CPU, bool) { return CPU{}, false }

// memoryStatusEx is MEMORYSTATUSEX of GlobalMemoryStatusEx.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var globalMemoryStatusEx = windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")

func (nativeWindows) Memory() (Memory, bool) {
	st := memoryStatusEx{Length: uint32(unsafe.Sizeof(memoryStatusEx{}))}
	if r, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&st))); r == 0 {
		return Memory{}, false
	}
	return physicalMemory(st.TotalPhys, st.AvailPhys)
}
