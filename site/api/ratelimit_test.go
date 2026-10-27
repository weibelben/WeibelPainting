package main

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter(5, time.Hour)
	start := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

	for i := range 5 {
		if !l.allow("1.2.3.4", start.Add(time.Duration(i)*time.Minute)) {
			t.Fatalf("request %d denied", i+1)
		}
	}
	if l.allow("1.2.3.4", start.Add(10*time.Minute)) {
		t.Fatal("6th request in an hour allowed")
	}
	if !l.allow("5.6.7.8", start.Add(10*time.Minute)) {
		t.Fatal("other IP denied")
	}
	// The first request ages out after an hour, freeing one slot.
	if !l.allow("1.2.3.4", start.Add(time.Hour+time.Second)) {
		t.Fatal("request after window denied")
	}
	if l.allow("1.2.3.4", start.Add(time.Hour+2*time.Second)) {
		t.Fatal("second request after window allowed; only one slot freed")
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		header, remote, want string
	}{
		{"203.0.113.7", "172.20.0.3:5555", "203.0.113.7"},
		{"2603:800c:4b00:188:79ef:38c0:d565:c081", "172.20.0.3:5555", "2603:800c:4b00:188::/64"},
		{"2603:800c:4b00:188:1::1", "172.20.0.3:5555", "2603:800c:4b00:188::/64"},
		{"", "192.0.2.1:5555", "192.0.2.1"},
	}
	for _, tt := range tests {
		r := httptest.NewRequest("POST", "/api/inquiries", nil)
		r.RemoteAddr = tt.remote
		if tt.header != "" {
			r.Header.Set("CF-Connecting-IP", tt.header)
		}
		if got := clientIP(r); got != tt.want {
			t.Errorf("clientIP(%q, %q) = %q, want %q", tt.header, tt.remote, got, tt.want)
		}
	}
}
