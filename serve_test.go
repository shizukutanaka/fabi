package main

import "testing"

func TestServeRequestAllowed(t *testing.T) {
	cases := []struct {
		name                                  string
		method, path, host, origin, fetchSite string
		want                                  bool
	}{
		{"cli curl", "GET", "/status", "127.0.0.1:11435", "", "", true},
		{"cli localhost", "POST", "/scan", "localhost:11435", "", "", true},
		{"no host (HTTP/1.0)", "GET", "/status", "", "", "", true},
		{"bare loopback host", "GET", "/status", "localhost", "", "", true},
		{"ui fetch same-origin", "GET", "/status", "127.0.0.1:11435", "", "same-origin", true},
		{"ui POST same-origin", "POST", "/unload", "127.0.0.1:11435", "http://127.0.0.1:11435", "same-origin", true},
		{"ui via localhost", "POST", "/scan", "localhost:11435", "http://localhost:11435", "same-origin", true},
		{"typed url", "GET", "/ledger", "127.0.0.1:11435", "", "none", true},
		{"ipv6 loopback", "GET", "/status", "[::1]:11435", "http://[::1]:11435", "same-origin", true},
		{"dns rebinding", "GET", "/ledger", "rebind.evil.example:11435", "", "", false},
		{"rebinding to ui", "GET", "/ui", "rebind.evil.example:11435", "", "", false},
		{"wrong port host", "GET", "/status", "127.0.0.1:8080", "", "", false},
		{"csrf pattern url", "POST", "/pattern", "127.0.0.1:11435", "https://evil.example", "cross-site", false},
		{"csrf no fetch-site", "POST", "/import", "127.0.0.1:11435", "https://evil.example", "", false},
		{"other local port", "POST", "/run", "127.0.0.1:11435", "http://127.0.0.1:3000", "same-site", false},
		{"opaque origin", "POST", "/unload", "127.0.0.1:11435", "null", "cross-site", false},
		{"img tag get", "GET", "/update", "127.0.0.1:11435", "", "cross-site", false},
		{"https origin spoof", "POST", "/scan", "127.0.0.1:11435", "https://127.0.0.1:11435", "", false},
		{"link to ui", "GET", "/ui", "127.0.0.1:11435", "", "cross-site", true},
	}
	for _, c := range cases {
		if got := serveRequestAllowed(c.method, c.path, c.host, c.origin, c.fetchSite); got != c.want {
			t.Errorf("%s: serveRequestAllowed(%q %q host=%q origin=%q site=%q) = %v, want %v",
				c.name, c.method, c.path, c.host, c.origin, c.fetchSite, got, c.want)
		}
	}
}
