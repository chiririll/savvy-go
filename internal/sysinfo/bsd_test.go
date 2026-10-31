package sysinfo

import (
	"encoding/binary"
	"testing"
)

func TestParseLoadavg(t *testing.T) {
	raw64 := make([]byte, 24)
	for i, v := range []uint32{2048, 1024, 512} {
		binary.LittleEndian.PutUint32(raw64[i*4:], v)
	}
	binary.LittleEndian.PutUint64(raw64[16:], 2048)
	got, ok := parseLoadavg(raw64, binary.LittleEndian)
	if !ok || got != [3]float64{1, 0.5, 0.25} {
		t.Fatalf("64-bit: got %v %v", got, ok)
	}

	raw32 := make([]byte, 16)
	copy(raw32, raw64[:12])
	binary.LittleEndian.PutUint32(raw32[12:], 2048)
	if got, ok := parseLoadavg(raw32, binary.LittleEndian); !ok || got != [3]float64{1, 0.5, 0.25} {
		t.Fatalf("32-bit: got %v %v", got, ok)
	}

	if _, ok := parseLoadavg(raw64[:8], binary.LittleEndian); ok {
		t.Error("short: want not ok")
	}
	if _, ok := parseLoadavg(make([]byte, 24), binary.LittleEndian); ok {
		t.Error("zero scale: want not ok")
	}
}
