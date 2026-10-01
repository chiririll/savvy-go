//go:build windows

package httpserver

import (
	"golang.org/x/sys/windows"
)

func diskUsage(path string) (total, free int64, ok bool) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, false
	}
	var freeBytes, totalBytes, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &freeBytes, &totalBytes, &totalFree); err != nil {
		return 0, 0, false
	}
	return int64(totalBytes), int64(freeBytes), true
}
