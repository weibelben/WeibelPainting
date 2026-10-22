package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// rateLimiter allows at most limit events per key within a sliding window.
// State is in memory, so it resets when the API restarts.
type rateLimiter struct {
	mu        sync.Mutex
	limit     int
	window    time.Duration
	hits      map[string][]time.Time
	lastSweep time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, hits: map[string][]time.Time{}}
}

// allow records an event for key and reports whether it is within the limit.
func (l *rateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	if now.Sub(l.lastSweep) > l.window {
		for k, times := range l.hits {
			if len(times) == 0 || !times[len(times)-1].After(cutoff) {
				delete(l.hits, k)
			}
		}
		l.lastSweep = now
	}

	times := l.hits[key]
	i := 0
	for i < len(times) && !times[i].After(cutoff) {
		i++
	}
	times = times[i:]
	if len(times) >= l.limit {
		l.hits[key] = times
		return false
	}
	l.hits[key] = append(times, now)
	return true
}

// clientIP returns the visitor's address. CF-Connecting-IP is trustworthy
// because the only public route in is the Cloudflare tunnel. IPv6 addresses
// are grouped by /64, since one visitor typically controls a whole /64.
func clientIP(r *http.Request) string {
	s := r.Header.Get("CF-Connecting-IP")
	if s == "" {
		s, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return s
	}
	if ip.To4() == nil {
		return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
	}
	return ip.String()
}
