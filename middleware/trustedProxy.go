package middleware

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"strings"
)

// TrustedProxy securely configures HTTP requests when behind a reverse proxy. How
// do you safely trust forwarded headers without creating security vulnerabilities?
//
// When your Go app sits behind nginx, the original client information (IP, protocol,
// host) gets lost because nginx becomes the direct client. Nginx forwards the real
// client info via headers like X-Forwarded-Proto and X-Forwarded-Host, but blindly
// trusting these headers is dangerous - any client could spoof them.
//
// Security Model:
//   - Only trusts headers from explicitly configured proxy IPs
//   - Validates source IP against trusted proxy list before processing headers
//   - Provides granular control over which headers to trust
//   - Safe by default (no headers trusted unless explicitly configured)
//
// Configuration via environment variables:
//
//	TRUSTED_PROXIES: Comma-separated list of trusted proxy IPs/CIDRs
//	                Examples: "127.0.0.1,192.168.1.0/24" or "10.0.0.0/8"
//	TRUST_PROXY_HEADERS: Comma-separated list of headers to trust
//	                    Examples: "proto,host" or "proto,host,port,for"
//
// Security considerations:
//   - Never set TRUSTED_PROXIES to "*" or "0.0.0.0/0" in production
//   - Only include your actual reverse proxy IPs
//   - Headers from untrusted IPs are completely ignored
func (a *Middleware) TrustedProxy(next http.Handler) http.Handler {
	// Parse trusted proxy configuration once at startup
	trustedProxies := parseTrustedProxies(os.Getenv("TRUSTED_PROXIES"))
	trustedHeaders := parseTrustedHeaders(os.Getenv("TRUST_PROXY_HEADERS"))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Extract client IP (could be from X-Real-IP or direct connection)
		clientIP := getClientIP(r)

		// Only process headers if request comes from a trusted proxy
		if isTrustedProxy(clientIP, trustedProxies) {
			// Process X-Forwarded-Proto if trusted
			if contains(trustedHeaders, "proto") {
				if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
					r.URL.Scheme = "https"
					r.TLS = &tls.ConnectionState{}
				}
			}

			// Process X-Forwarded-Host if trusted
			if contains(trustedHeaders, "host") {
				if host := r.Header.Get("X-Forwarded-Host"); host != "" {
					r.Host = host
					r.URL.Host = host
				}
			}

		}

		// If not from trusted proxy, ignore all headers (secure default)
		next.ServeHTTP(w, r)
	})
}

// parseTrustedProxies converts environment string to list of trusted networks
func parseTrustedProxies(proxyList string) []*net.IPNet {
	if proxyList == "" {
		return nil // No proxies trusted by default
	}

	var networks []*net.IPNet
	proxies := strings.Split(proxyList, ",")

	for _, proxy := range proxies {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}

		// Handle single IP (add /32 or /128)
		if !strings.Contains(proxy, "/") {
			if strings.Contains(proxy, ":") {
				proxy += "/128" // IPv6
			} else {
				proxy += "/32" // IPv4
			}
		}

		_, network, err := net.ParseCIDR(proxy)
		if err == nil {
			networks = append(networks, network)
		}
	}

	return networks
}

// parseTrustedHeaders converts environment string to list of trusted headers
func parseTrustedHeaders(headerList string) []string {
	if headerList == "" {
		return []string{"proto", "host"} // Safe defaults
	}

	headers := strings.Split(headerList, ",")
	for i, header := range headers {
		headers[i] = strings.TrimSpace(header)
	}
	return headers
}

// getClientIP extracts the real client IP, handling various proxy scenarios
func getClientIP(r *http.Request) string {
	// Try X-Real-IP first (most reliable from reverse proxy)
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Try X-Forwarded-For (could be comma-separated list)
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// Take the first IP (original client)
		if parts := strings.Split(ip, ","); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Fall back to direct connection IP
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return ip
	}

	return r.RemoteAddr
}

// isTrustedProxy checks if the given IP is in the trusted proxy list
func isTrustedProxy(ip string, trustedNetworks []*net.IPNet) bool {
	if len(trustedNetworks) == 0 {
		return false // No proxies trusted
	}

	clientIP := net.ParseIP(ip)
	if clientIP == nil {
		return false
	}

	for _, network := range trustedNetworks {
		if network.Contains(clientIP) {
			return true
		}
	}

	return false
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
