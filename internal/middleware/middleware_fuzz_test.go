// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// FuzzExtractTunnelIdFromPath tests tunnel ID extraction from URL paths
func FuzzExtractTunnelIdFromPath(f *testing.F) {
	// Add seed corpus
	f.Add("/kubernetes/tunnel-123/api/v1/pods")
	f.Add("/kubernetes//api/v1/pods")
	f.Add("/kubernetes")
	f.Add("/api/v1/pods")
	f.Add("")
	f.Add("/kubernetes/tunnel-with-uuid-12345678-1234-1234-1234-123456789abc-cluster/resources")
	f.Add("/kubernetes/tunnel!/path")

	f.Fuzz(func(t *testing.T, path string) {
		// Skip empty or invalid paths that would cause httptest.NewRequest to panic
		if path == "" || path[0] != '/' {
			return
		}

		// Skip paths with control characters (0x00-0x1F and 0x7F) or spaces
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

		// Call extractTunnelId - should not panic
		tunnelID, err := extractTunnelId(req)

		// Valid paths should have format: /kubernetes/{tunnelId}/...
		if !strings.Contains(path, "/kubernetes/") {
			if err == nil {
				t.Logf("Expected error for invalid path: %s", path)
			}
		} else {
			if err == nil && tunnelID == "" {
				t.Error("Expected non-empty tunnel ID for valid path")
			}
		}
	})
}

// FuzzExtractProjectIdFromTunnel tests project ID extraction from tunnel strings
func FuzzExtractProjectIdFromTunnel(f *testing.F) {
	// Add seed corpus with valid UUID formats
	f.Add("12345678-1234-1234-1234-123456789abc-cluster-name")
	f.Add("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee-test")
	f.Add("") // Empty string
	f.Add("invalid-format")
	f.Add("12345678-1234-1234-1234") // Too short
	f.Add("a-b-c-d-e-f-g-h-i-j-k")   // Many segments

	f.Fuzz(func(t *testing.T, tunnelID string) {
		// Call extractProjectIdFromTunnel - should not panic
		projectID, err := extractProjectIdFromTunnel(tunnelID)

		// Valid tunnel IDs need at least 6 segments (UUID is 5 segments)
		segments := strings.Split(tunnelID, "-")
		if len(segments) < 6 {
			if err == nil {
				t.Errorf("Expected error for tunnel ID with %d segments: %s", len(segments), tunnelID)
			}
		} else {
			if err == nil && projectID == "" {
				t.Error("Expected non-empty project ID for valid tunnel format")
			}
		}
	})
}

// FuzzURLPathSegments tests various URL path segment patterns
func FuzzURLPathSegments(f *testing.F) {
	f.Add("/segment1/segment2/segment3")
	f.Add("//double/slash")
	f.Add("/")
	f.Add("")
	f.Add("/single")
	f.Add("/a/b/c/d/e/f/g/h/i/j") // Many segments
	f.Add("/kubernetes/12345678-1234-1234-1234-123456789abc-cluster/api/v1/namespaces/default/pods")

	f.Fuzz(func(t *testing.T, path string) {
		// Skip empty paths and paths that don't start with /
		if path == "" || path[0] != '/' {
			return
		}

		// Skip paths with control characters (0x00-0x1F and 0x7F) or spaces
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

		// Test path parsing
		segments := strings.Split(req.URL.Path, "/")
		_ = len(segments)

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}

// FuzzHTTPRequestMethods tests various HTTP methods with middleware
func FuzzHTTPRequestMethods(f *testing.F) {
	f.Add("GET", "/kubernetes/tunnel-123/api/v1/pods", "application/json")
	f.Add("POST", "/kubernetes/tunnel-456/api/v1/pods", "application/json")
	f.Add("PUT", "/kubernetes/tunnel-789/api/v1/pods/test", "application/json")
	f.Add("DELETE", "/kubernetes/tunnel-abc/api/v1/pods/test", "")
	f.Add("PATCH", "/kubernetes/tunnel-def/api/v1/pods/test", "application/merge-patch+json")

	f.Fuzz(func(t *testing.T, method string, path string, contentType string) {
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

		req := httptest.NewRequest(method, path, nil)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		// Test request doesn't panic
		_ = req.Method
		_ = req.URL.Path
		_ = req.Header.Get("Content-Type")

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}

// FuzzQueryParameters tests URL query parameter handling
func FuzzQueryParameters(f *testing.F) {
	f.Add("param1", "value1", "param2", "value2")
	f.Add("", "", "", "")
	f.Add("key", "value=with=equals", "", "")
	f.Add("special", "!@#$%^&*()", "", "")

	f.Fuzz(func(t *testing.T, key1 string, val1 string, key2 string, val2 string) {
		path := "/kubernetes/tunnel-123/api/v1/pods"

		// Build query string with URL encoding to prevent malformed URLs
		query := ""
		if key1 != "" && val1 != "" {
			// Skip control characters (0x00-0x1F and 0x7F) and space
			if strings.ContainsFunc(key1+val1, func(r rune) bool {
				return r < 0x20 || r == 0x7F || r == ' '
			}) {
				return
			}
			query = "?" + key1 + "=" + val1
		}
		if key2 != "" && val2 != "" {
			if strings.ContainsFunc(key2+val2, func(r rune) bool {
				return r < 0x20 || r == 0x7F || r == ' '
			}) {
				return
			}
			if query == "" {
				query = "?" + key2 + "=" + val2
			} else {
				query += "&" + key2 + "=" + val2
			}
		}

		fullPath := path + query
		req := httptest.NewRequest("GET", fullPath, nil)

		// Test query parameter handling
		_ = req.URL.Query()
		_ = req.URL.RawQuery

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}

// FuzzContentTypes tests various Content-Type header values
func FuzzContentTypes(f *testing.F) {
	f.Add("application/json")
	f.Add("application/xml")
	f.Add("text/plain")
	f.Add("multipart/form-data")
	f.Add("application/x-www-form-urlencoded")
	f.Add("")
	f.Add("invalid/content-type")
	f.Add("application/json; charset=utf-8")

	f.Fuzz(func(t *testing.T, contentType string) {
		req := httptest.NewRequest("POST", "/kubernetes/tunnel-123/api/v1/pods", nil)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		// Test Content-Type handling
		ct := req.Header.Get("Content-Type")
		_ = ct

		// Test tunnel ID extraction
		_, _ = extractTunnelId(req)
	})
}

// FuzzUUIDFormats tests various UUID format patterns in tunnel IDs
func FuzzUUIDFormats(f *testing.F) {
	// Valid UUID v4 format examples
	f.Add(byte(0x12), byte(0x34), byte(0x56), byte(0x78), byte(0x90), byte(0xab), byte(0xcd), byte(0xef))
	f.Add(byte(0xff), byte(0xff), byte(0xff), byte(0xff), byte(0x00), byte(0x00), byte(0x00), byte(0x00))
	f.Add(byte(0x00), byte(0x00), byte(0x00), byte(0x00), byte(0x00), byte(0x00), byte(0x00), byte(0x00))

	f.Fuzz(func(t *testing.T, b0, b1, b2, b3, b4, b5, b6, b7 byte) {
		// Create a UUID-like string from bytes
		uuid := fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-",
			b0, b1, b2, b3, b4, b5, b6, b7)
		uuid += fmt.Sprintf("%02x%02x-%02x%02x%02x%02x%02x%02x-cluster",
			b0, b1, b2, b3, b4, b5, b6, b7)

		// Test project ID extraction
		_, _ = extractProjectIdFromTunnel(uuid)
	})
}

// FuzzResponseWriter tests response writer operations
func FuzzResponseWriter(f *testing.F) {
	f.Add(200, "OK", "application/json")
	f.Add(404, "Not Found", "text/plain")
	f.Add(500, "Internal Server Error", "application/json")
	f.Add(0, "", "")

	f.Fuzz(func(t *testing.T, statusCode int, statusText string, contentType string) {
		// Skip invalid status codes - HTTP status codes must be between 100 and 599
		if statusCode < 100 || statusCode > 599 {
			return
		}

		rr := httptest.NewRecorder()

		if contentType != "" {
			rr.Header().Set("Content-Type", contentType)
		}

		rr.WriteHeader(statusCode)

		if statusText != "" {
			_, _ = rr.WriteString(statusText)
		}

		// Test response recorder
		_ = rr.Code
		_ = rr.Body.String()
	})
}
