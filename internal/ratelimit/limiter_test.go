package ratelimit

import (
	"testing"
	"time"
)

func newTest(perMinute int) (*Limiter, *time.Time) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := New(perMinute)
	l.now = func() time.Time { return now }
	return l, &now
}

func TestBurstOverLimitIsRefused(t *testing.T) {
	l, _ := newTest(3)
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d refused inside the limit", i+1)
		}
	}
	ok, wait := l.Allow("a")
	if ok {
		t.Fatal("fourth request passed")
	}
	if wait <= 0 || wait > 20*time.Second {
		t.Errorf("retry after %v, want one token's refill (20s)", wait)
	}
}

func TestKeysAreIndependent(t *testing.T) {
	l, _ := newTest(1)
	l.Allow("a")
	if ok, _ := l.Allow("b"); !ok {
		t.Error("another key was refused")
	}
}

func TestPassesAgainAfterWindow(t *testing.T) {
	l, now := newTest(2)
	l.Allow("a")
	l.Allow("a")
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("over the limit passed")
	}
	*now = now.Add(31 * time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Error("one token should have refilled after half a minute")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Error("only one token should have refilled")
	}
	*now = now.Add(time.Hour)
	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Errorf("full burst %d refused after a long pause", i+1)
		}
	}
}

func TestZeroLimitIsOff(t *testing.T) {
	l, _ := newTest(0)
	for i := 0; i < 1000; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatal("a disabled limiter refused")
		}
	}
}

func TestIdleKeysAreForgotten(t *testing.T) {
	l, now := newTest(5)
	for i := 0; i < 100; i++ {
		l.Allow(string(rune('a' + i)))
	}
	*now = now.Add(time.Hour)
	l.Allow("fresh")
	if n := l.Len(); n != 1 {
		t.Errorf("%d keys kept after an idle hour, want 1", n)
	}
}
