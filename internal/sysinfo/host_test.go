package sysinfo

import "testing"

func TestLoadPercent(t *testing.T) {
	c := CPU{Load: [3]float64{3.9, 3.5, 2.8}, Cores: 4}
	if pct, ok := c.LoadPercent(); !ok || pct != 97.5 {
		t.Fatalf("got %v %v", pct, ok)
	}
	if pct, ok := (CPU{Load: [3]float64{3.9}, Cores: 2}).LoadPercent(); !ok || pct != 195 {
		t.Fatalf("overload: got %v %v", pct, ok)
	}
	if pct, ok := (CPU{Load: [3]float64{0.5}, Cores: 0.5}).LoadPercent(); !ok || pct != 100 {
		t.Fatalf("fractional quota: got %v %v", pct, ok)
	}
	if _, ok := (CPU{Load: [3]float64{1}}).LoadPercent(); ok {
		t.Fatal("no cores: want not ok")
	}
}

func TestUsedPercent(t *testing.T) {
	if pct, ok := (Memory{Total: 2000, Used: 1200}).UsedPercent(); !ok || pct != 60 {
		t.Fatalf("got %v %v", pct, ok)
	}
	if pct, ok := (Memory{Total: 3, Used: 1}).UsedPercent(); !ok || pct != 33.3 {
		t.Fatalf("rounding: got %v %v", pct, ok)
	}
	if _, ok := (Memory{}).UsedPercent(); ok {
		t.Fatal("no total: want not ok")
	}
}

func TestPhysicalMemory(t *testing.T) {
	m, ok := physicalMemory(1000, 400)
	if !ok || m != (Memory{Total: 1000, Used: 600, Free: 400, Source: "host"}) {
		t.Fatalf("got %+v %v", m, ok)
	}
	if m, _ := physicalMemory(1000, 5000); m.Used != 0 || m.Free != 1000 {
		t.Fatalf("available above total: got %+v", m)
	}
	if _, ok := physicalMemory(0, 0); ok {
		t.Fatal("no total: want not ok")
	}
}
