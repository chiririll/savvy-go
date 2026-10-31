package sysinfo

import "encoding/binary"

// parseLoadavg reads struct loadavg of the BSDs: three uint32 fixed-point
// values, padding, then the scale (a C long, 8 bytes on 64-bit systems).
func parseLoadavg(raw []byte, order binary.ByteOrder) (load [3]float64, ok bool) {
	var scale float64
	switch {
	case len(raw) >= 24:
		scale = float64(order.Uint64(raw[16:24]))
	case len(raw) >= 16:
		scale = float64(order.Uint32(raw[12:16]))
	default:
		return load, false
	}
	if scale <= 0 {
		return load, false
	}
	for i := range load {
		load[i] = float64(order.Uint32(raw[i*4:])) / scale
	}
	return load, true
}
