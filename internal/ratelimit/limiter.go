// Package ratelimit is an in-memory token-bucket limiter. The process is one Go binary, so
// the counters need no shared store; a restart forgets them.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter lets each key make perMinute requests in a burst and refills at that rate.
type Limiter struct {
	perMinute int
	now       func() time.Time

	mu      sync.Mutex
	buckets map[string]bucket
	swept   time.Time
}

type bucket struct {
	tokens float64
	at     time.Time
}

// New returns a limiter of perMinute requests per key. Zero or less lets everything pass.
func New(perMinute int) *Limiter {
	return &Limiter{perMinute: perMinute, now: time.Now, buckets: map[string]bucket{}}
}

// Allow takes a token for key. When none is left it returns false and how long until one is.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	if l == nil || l.perMinute <= 0 {
		return true, 0
	}
	now := l.now()
	rate := float64(l.perMinute) / float64(time.Minute) // tokens per nanosecond

	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)

	b, ok := l.buckets[key]
	if !ok {
		b = bucket{tokens: float64(l.perMinute), at: now}
	}
	b.tokens = min(float64(l.perMinute), b.tokens+float64(now.Sub(b.at))*rate)
	b.at = now
	if b.tokens < 1 {
		l.buckets[key] = b
		return false, time.Duration((1 - b.tokens) / rate)
	}
	b.tokens--
	l.buckets[key] = b
	return true, 0
}

// sweep drops keys that have been idle long enough to be full again, at most once a minute,
// so the map does not grow with every address that ever called.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.swept) < time.Minute {
		return
	}
	l.swept = now
	for k, b := range l.buckets {
		if now.Sub(b.at) >= time.Minute {
			delete(l.buckets, k)
		}
	}
}

// Len is how many keys are being tracked.
func (l *Limiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}
