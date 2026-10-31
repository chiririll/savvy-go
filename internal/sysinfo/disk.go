package sysinfo

import (
	"os"
	"path/filepath"
)

// volumeSpace is the size and free space of the volume holding path, or of its
// nearest existing parent.
func volumeSpace(path string) (total, free int64, ok bool) {
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
