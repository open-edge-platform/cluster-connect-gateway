// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/open-edge-platform/cluster-connect-gateway/internal/agent"
)

// FuzzExtractTunnelId tests tunnel ID extraction from HTTP headers
func FuzzExtractTunnelId(f *testing.F) {
	// Add seed corpus
	f.Add("tunnel-123")
	f.Add("")
	f.Add("tunnel-with-special-chars-!@#$%^&*()")
	f.Add(string(make([]byte, 500))) // Very long tunnel ID
	f.Add("tunnel\nwith\nnewlines")
	f.Add("tunnel\x00with\x00nulls")

	f.Fuzz(func(t *testing.T, tunnelID string) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(agent.TunnelIdHeader, tunnelID)

		// Call extractTunnelId - should not panic
		id, err := extractTunnelId(req)

		if tunnelID == "" {
			if err == nil {
				t.Error("Expected error for empty tunnel ID")
			}
		} else {
			if err != nil {
				t.Logf("Error extracting tunnel ID: %v", err)
			}
			if id != tunnelID {
				t.Errorf("Expected tunnel ID %q, got %q", tunnelID, id)
			}
		}
	})
}

// FuzzAuthorizationHeaders tests various authorization header combinations
func FuzzAuthorizationHeaders(f *testing.F) {
	// Add seed corpus
	f.Add("Bearer token123", "tunnel-1", "secret-token-abc")
	f.Add("", "", "")
	f.Add("Basic dXNlcjpwYXNz", "tunnel-2", "token-xyz")
	f.Add("Bearer ", "tunnel-3", "")
	f.Add("InvalidScheme token", "tunnel-4", "valid-token")

	f.Fuzz(func(t *testing.T, authHeader string, tunnelID string, tokenHeader string) {
		req := httptest.NewRequest("GET", "/test", nil)

		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
		if tunnelID != "" {
			req.Header.Set(agent.TunnelIdHeader, tunnelID)
		}
		if tokenHeader != "" {
			req.Header.Set(agent.TokenHeader, tokenHeader)
		}

		// Test header retrieval - should not panic
		_ = req.Header.Get("Authorization")
		_ = req.Header.Get(agent.TunnelIdHeader)
		_ = req.Header.Get(agent.TokenHeader)

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}

// FuzzHTTPMethods tests different HTTP methods with authorization
func FuzzHTTPMethods(f *testing.F) {
	f.Add("GET", "/api/v1/resource")
	f.Add("POST", "/api/v1/create")
	f.Add("PUT", "/api/v1/update")
	f.Add("DELETE", "/api/v1/delete")
	f.Add("PATCH", "/api/v1/patch")
	f.Add("HEAD", "/api/v1/head")
	f.Add("OPTIONS", "/api/v1/options")
	f.Add("INVALID", "/api/v1/test")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, method string, path string) {
		// Skip invalid inputs that would cause httptest.NewRequest to panic
		if method == "" || strings.TrimSpace(method) == "" {
			return
		}

		// HTTP methods must be valid tokens: printable ASCII (0x21-0x7E) except separators
		// Skip methods with invalid characters per RFC 7230
		if strings.ContainsFunc(method, func(r rune) bool {
			return r < 0x21 || r > 0x7E || strings.ContainsRune("\"'(),/:;<=>?@[\\]{}!", r)
		}) {
			return
		}

		if path == "" {
			path = "/"
		}

		// Skip paths that are not valid URIs for requests
		// Must start with / or be a valid absolute URL with a proper scheme
		if !strings.HasPrefix(path, "/") {
			// If it contains "://", validate it's a proper URL with a scheme
			if strings.Contains(path, "://") {
				parsedURL, err := url.Parse(path)
				if err != nil || parsedURL.Scheme == "" {
					return
				}
			} else {
				// Not a path starting with / and not a URL
				return
			}
		}

		// Skip paths with control characters or spaces
		if strings.Contains(path, " ") || strings.ContainsFunc(path, func(r rune) bool {
			return r < 0x20 || r == 0x7F
		}) {
			return
		}

		// Skip paths with invalid URL escapes
		if _, err := url.PathUnescape(path); err != nil {
			return
		}

		// Create request with fuzzy method and path
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set(agent.TunnelIdHeader, "test-tunnel")

		// Test that request handling doesn't panic
		_ = req.Method
		_ = req.URL.Path

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}

// FuzzHeaderKeys tests various header key formats
func FuzzHeaderKeys(f *testing.F) {
	f.Add("X-Custom-Header", "value1")
	f.Add("x-lowercase-header", "value2")
	f.Add("UPPERCASE-HEADER", "value3")
	f.Add("Header-With-Numbers-123", "value4")
	f.Add("", "")
	f.Add("Invalid Header!", "value5")
	f.Add("Header\nWith\nNewlines", "value6")

	f.Fuzz(func(t *testing.T, headerKey string, headerValue string) {
		req := httptest.NewRequest("GET", "/test", nil)

		// Set fuzzy header
		if headerKey != "" {
			req.Header.Set(headerKey, headerValue)
		}

		// Always set required tunnel ID header
		req.Header.Set(agent.TunnelIdHeader, "test-tunnel")

		// Test header operations don't panic
		_ = req.Header.Get(headerKey)

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}

// FuzzURLPaths tests various URL path formats
func FuzzURLPaths(f *testing.F) {
	f.Add("/api/v1/clusters")
	f.Add("/")
	f.Add("")
	f.Add("//double/slash")
	f.Add("/path/with/../../traversal")
	f.Add("/path?query=value&other=123")
	f.Add("/path#fragment")
	f.Add(string(make([]byte, 2000))) // Very long path

	f.Fuzz(func(t *testing.T, path string) {
		// Skip empty paths and paths that don't start with /
		if path == "" || (len(path) > 0 && path[0] != '/') {
			return
		}

		// Skip paths with control characters (0x00-0x1F and 0x7F) or spaces that break URL parsing
		if strings.ContainsFunc(path, func(r rune) bool {
			return r < 0x20 || r == 0x7F || r == ' '
		}) {
			return
		}

		// Skip paths with invalid URL escapes
		if strings.Contains(path, "%") {
			if _, err := url.PathUnescape(path); err != nil {
				return
			}
		}

		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set(agent.TunnelIdHeader, "test-tunnel")

		// Test URL parsing doesn't panic
		_ = req.URL.Path
		_ = req.URL.RawQuery

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}

// FuzzMultipleHeaders tests requests with multiple header values
func FuzzMultipleHeaders(f *testing.F) {
	f.Add("value1", "value2", "value3")
	f.Add("", "", "")
	f.Add("single", "", "")

	f.Fuzz(func(t *testing.T, val1 string, val2 string, val3 string) {
		req := httptest.NewRequest("GET", "/test", nil)

		// Add multiple values for the same header
		if val1 != "" {
			req.Header.Add("X-Custom-Header", val1)
		}
		if val2 != "" {
			req.Header.Add("X-Custom-Header", val2)
		}
		if val3 != "" {
			req.Header.Add("X-Custom-Header", val3)
		}

		req.Header.Set(agent.TunnelIdHeader, "test-tunnel")

		// Test header retrieval
		_ = req.Header.Get("X-Custom-Header")
		values := req.Header.Values("X-Custom-Header")
		_ = len(values)

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}

// FuzzTokenFormats tests various token format patterns
func FuzzTokenFormats(f *testing.F) {
	// Add various JWT-like and token formats
	f.Add("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")
	f.Add("simple-token-123")
	f.Add("")
	f.Add("token-with-special-!@#$%^&*()")
	f.Add(string(make([]byte, 1000))) // Very long token

	f.Fuzz(func(t *testing.T, token string) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(agent.TunnelIdHeader, "test-tunnel")
		req.Header.Set(agent.TokenHeader, token)

		// Test token retrieval
		retrievedToken := req.Header.Get(agent.TokenHeader)
		if retrievedToken != token {
			t.Errorf("Expected token %q, got %q", token, retrievedToken)
		}

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}
