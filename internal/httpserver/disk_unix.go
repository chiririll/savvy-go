//go:build !windows

package httpserver

import "golang.org/x/sys/unix"

func diskUsage(path string) (total, free int64, ok bool) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, 0, false
	}
	bs := int64(st.Bsize)
	return bs * int64(st.Blocks), bs * int64(st.Bavail), true
}
