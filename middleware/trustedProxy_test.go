package middleware

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestTrustedProxy_TrustedIPWithHTTPS(t *testing.T) {
	// Set environment variables
	os.Setenv("TRUSTED_PROXIES", "127.0.0.1")
	os.Setenv("TRUST_PROXY_HEADERS", "proto,host")
	defer func() {
		os.Unsetenv("TRUSTED_PROXIES")
		os.Unsetenv("TRUST_PROXY_HEADERS")
	}()

	m := &Middleware{}
	var capturedRequest *http.Request

	handler := m.TrustedProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "example.com")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Verify HTTPS was set
	if capturedRequest.URL.Scheme != "https" {
		t.Errorf("Expected scheme 'https', got '%s'", capturedRequest.URL.Scheme)
	}

	if capturedRequest.TLS == nil {
		t.Error("Expected TLS to be set for HTTPS")
	}

	if capturedRequest.Host != "example.com" {
		t.Errorf("Expected host 'example.com', got '%s'", capturedRequest.Host)
	}
}

func TestTrustedProxy_UntrustedIPIgnored(t *testing.T) {
	os.Setenv("TRUSTED_PROXIES", "127.0.0.1")
	os.Setenv("TRUST_PROXY_HEADERS", "proto,host")
	defer func() {
		os.Unsetenv("TRUSTED_PROXIES")
		os.Unsetenv("TRUST_PROXY_HEADERS")
	}()

	m := &Middleware{}
	var capturedRequest *http.Request

	handler := m.TrustedProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	req.RemoteAddr = "192.168.1.100:54321" // Untrusted IP
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "malicious.com")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Headers should be ignored
	if capturedRequest.URL.Scheme == "https" {
		t.Error("Expected scheme to not be set from untrusted proxy")
	}

	if capturedRequest.TLS != nil {
		t.Error("Expected TLS to not be set from untrusted proxy")
	}

	if capturedRequest.Host == "malicious.com" {
		t.Error("Expected host to not be set from untrusted proxy")
	}
}

func TestTrustedProxy_CIDRRange(t *testing.T) {
	os.Setenv("TRUSTED_PROXIES", "192.168.1.0/24")
	os.Setenv("TRUST_PROXY_HEADERS", "proto")
	defer func() {
		os.Unsetenv("TRUSTED_PROXIES")
		os.Unsetenv("TRUST_PROXY_HEADERS")
	}()

	m := &Middleware{}
	var capturedRequest *http.Request

	handler := m.TrustedProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	req.RemoteAddr = "192.168.1.50:12345" // Within CIDR range
	req.Header.Set("X-Forwarded-Proto", "https")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if capturedRequest.URL.Scheme != "https" {
		t.Errorf("Expected scheme 'https' from trusted CIDR range, got '%s'", capturedRequest.URL.Scheme)
	}
}

func TestTrustedProxy_SelectiveHeaders(t *testing.T) {
	os.Setenv("TRUSTED_PROXIES", "127.0.0.1")
	os.Setenv("TRUST_PROXY_HEADERS", "proto") // Only trust proto, not host
	defer func() {
		os.Unsetenv("TRUSTED_PROXIES")
		os.Unsetenv("TRUST_PROXY_HEADERS")
	}()

	m := &Middleware{}
	var capturedRequest *http.Request

	handler := m.TrustedProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "example.com")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Proto should be set
	if capturedRequest.URL.Scheme != "https" {
		t.Errorf("Expected scheme 'https', got '%s'", capturedRequest.URL.Scheme)
	}

	// Host should NOT be set (not in trusted headers)
	if capturedRequest.Host == "example.com" {
		t.Error("Expected host to not be set when not in trusted headers")
	}
}

func TestTrustedProxy_NoConfiguration(t *testing.T) {
	// No environment variables set
	m := &Middleware{}
	var capturedRequest *http.Request

	handler := m.TrustedProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequest = r
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "http://localhost/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-Proto", "https")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// Should not trust any headers when no proxies configured
	if capturedRequest.URL.Scheme == "https" {
		t.Error("Expected scheme to not be set when no proxies configured")
	}
}

func TestTrustedProxy_MultipleProxies(t *testing.T) {
	os.Setenv("TRUSTED_PROXIES", "127.0.0.1,10.0.0.1")
	os.Setenv("TRUST_PROXY_HEADERS", "proto")
	defer func() {
		os.Unsetenv("TRUSTED_PROXIES")
		os.Unsetenv("TRUST_PROXY_HEADERS")
	}()

	m := &Middleware{}

	testCases := []struct {
		name        string
		remoteAddr  string
		shouldTrust bool
	}{
		{"first proxy", "127.0.0.1:12345", true},
		{"second proxy", "10.0.0.1:12345", true},
		{"untrusted IP", "192.168.1.1:12345", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedRequest *http.Request

			handler := m.TrustedProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequest = r
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "http://localhost/test", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Header.Set("X-Forwarded-Proto", "https")

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if tc.shouldTrust {
				if capturedRequest.URL.Scheme != "https" {
					t.Errorf("Expected scheme 'https' from trusted proxy %s", tc.remoteAddr)
				}
			} else {
				if capturedRequest.URL.Scheme == "https" {
					t.Errorf("Expected scheme to not be set from untrusted proxy %s", tc.remoteAddr)
				}
			}
		})
	}
}

func TestParseTrustedProxies(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"single IP", "127.0.0.1", 1},
		{"multiple IPs", "127.0.0.1,192.168.1.1", 2},
		{"CIDR range", "192.168.1.0/24", 1},
		{"mixed", "127.0.0.1,192.168.1.0/24,10.0.0.1", 3},
		{"with spaces", "127.0.0.1, 192.168.1.1 , 10.0.0.1", 3},
		{"invalid IP", "invalid-ip", 0},
		{"IPv6", "::1,fe80::/10", 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			networks := parseTrustedProxies(tc.input)
			if len(networks) != tc.expected {
				t.Errorf("Expected %d networks, got %d", tc.expected, len(networks))
			}
		})
	}
}

func TestParseTrustedHeaders(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty uses defaults", "", []string{"proto", "host"}},
		{"single header", "proto", []string{"proto"}},
		{"multiple headers", "proto,host", []string{"proto", "host"}},
		{"with spaces", "proto, host , for", []string{"proto", "host", "for"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := parseTrustedHeaders(tc.input)
			if len(headers) != len(tc.expected) {
				t.Errorf("Expected %d headers, got %d", len(tc.expected), len(headers))
				return
			}
			for i, h := range headers {
				if h != tc.expected[i] {
					t.Errorf("Expected header %s at index %d, got %s", tc.expected[i], i, h)
				}
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	testCases := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "X-Real-IP priority",
			remoteAddr: "192.168.1.1:12345",
			headers: map[string]string{
				"X-Real-IP":       "203.0.113.1",
				"X-Forwarded-For": "203.0.113.2",
			},
			expected: "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For fallback",
			remoteAddr: "192.168.1.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 192.168.1.1",
			},
			expected: "203.0.113.1",
		},
		{
			name:       "RemoteAddr fallback",
			remoteAddr: "203.0.113.1:12345",
			headers:    map[string]string{},
			expected:   "203.0.113.1",
		},
		{
			name:       "IPv6",
			remoteAddr: "[::1]:12345",
			headers:    map[string]string{},
			expected:   "::1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tc.remoteAddr

			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			result := getClientIP(req)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestIsTrustedProxy(t *testing.T) {
	networks := parseTrustedProxies("127.0.0.1,192.168.1.0/24")

	testCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"localhost trusted", "127.0.0.1", true},
		{"within CIDR", "192.168.1.50", true},
		{"outside CIDR", "192.168.2.50", false},
		{"invalid IP", "not-an-ip", false},
		{"different network", "10.0.0.1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isTrustedProxy(tc.ip, networks)
			if result != tc.expected {
				t.Errorf("Expected %t for IP %s, got %t", tc.expected, tc.ip, result)
			}
		})
	}
}

func TestIsTrustedProxy_EmptyList(t *testing.T) {
	result := isTrustedProxy("127.0.0.1", []*net.IPNet{})
	if result {
		t.Error("Expected false when no proxies are trusted")
	}
}
