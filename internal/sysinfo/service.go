// Package sysinfo reads facts about the machine the server runs on: disk space,
// load average and memory.
package sysinfo

// Service is what the rest of the app asks about the machine: disk space, and
// load and memory through the collector of this platform. The zero Service
// knows no load or memory.
type Service struct {
	Collector Collector
}

// VolumeSpace is the size and free space of the volume holding path, or of its
// nearest existing parent.
func (Service) VolumeSpace(path string) (total, free int64, ok bool) { return volumeSpace(path) }

func (s Service) CPU() (CPU, bool) {
	if s.Collector == nil {
		return CPU{}, false
	}
	return s.Collector.CPU()
}

func (s Service) Memory() (Memory, bool) {
	if s.Collector == nil {
		return Memory{}, false
	}
	return s.Collector.Memory()
}
