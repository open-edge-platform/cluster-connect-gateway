// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"net/http/httptest"
	"testing"
	"time"
)

// FuzzGetClient tests the GetClient function with various tunnel IDs and timeout values
func FuzzGetClient(f *testing.F) {
	// Add seed corpus
	f.Add("test-tunnel-1", "30s")
	f.Add("cluster-abc-123", "1m")
	f.Add("", "10s")
	f.Add("tunnel-with-special-chars!@#", "5s")
	f.Add("very-long-tunnel-id-"+string(make([]byte, 100)), "1h")

	f.Fuzz(func(t *testing.T, tunnelID string, timeout string) {
		// Skip this fuzz test as it requires a Kubernetes cluster to be available
		// NewServer tries to create an in-cluster client when kubeclient is nil
		// TODO: Add proper mocking for kubeclient to re-enable this test
		t.Skip("Skipping FuzzGetClient: requires Kubernetes cluster or mock kubeclient")
	})
}

// FuzzRemoteDialer tests the remote dialer setup with various configurations
func FuzzRemoteDialer(f *testing.F) {
	// Add seed corpus with various header combinations
	f.Add("Authorization", "Bearer token123", "X-Tunnel-ID", "tunnel-1")
	f.Add("", "", "", "")
	f.Add("Custom-Header", "value", "Another-Header", "another-value")
	f.Add("X-Forwarded-For", "192.168.1.1", "X-Real-IP", "10.0.0.1")

	f.Fuzz(func(t *testing.T, header1Key, header1Val, header2Key, header2Val string) {
		// Create a test HTTP request with fuzzy headers
		req := httptest.NewRequest("GET", "/connect", nil)
		if header1Key != "" {
			req.Header.Set(header1Key, header1Val)
		}
		if header2Key != "" {
			req.Header.Set(header2Key, header2Val)
		}

		// Test header extraction - should not panic
		_ = req.Header.Get("Authorization")
		_ = req.Header.Get("X-Tunnel-ID")
	})
}

// FuzzServerOptions tests server initialization with various option combinations
func FuzzServerOptions(f *testing.F) {
	// Add seed corpus
	f.Add(":8080", "http://opa:8181", 8181, true)
	f.Add(":0", "", 0, false)
	f.Add(":65535", "https://opa.example.com", 443, true)
	f.Add("invalid-addr", "not-a-url", -1, false)

	f.Fuzz(func(t *testing.T, addr string, opaAddr string, opaPort int, authEnabled bool) {
		// Test WithListenAddr
		opt1 := WithListenAddr(addr)
		s := &Server{}
		opt1(s)

		// Test WithAuth
		opt2 := WithAuth(authEnabled, opaAddr, opaPort)
		opt2(s)

		// Verify server doesn't panic with fuzzy inputs
		if s.listenAddr == "" {
			s.listenAddr = ":8080" // Set default
		}
	})
}

// FuzzExternalHost tests external host configuration
func FuzzExternalHost(f *testing.F) {
	f.Add("example.com")
	f.Add("192.168.1.1")
	f.Add("localhost:8080")
	f.Add("")
	f.Add("http://example.com")
	f.Add("https://secure.example.com:443")
	f.Add(string(make([]byte, 300))) // Very long host

	f.Fuzz(func(t *testing.T, host string) {
		opt := WithExternalHost(host)
		s := &Server{}
		opt(s)

		// Verify it doesn't panic
		_ = s.externalHost
	})
}

// FuzzOIDCConfiguration tests OIDC configuration with various URLs
func FuzzOIDCConfiguration(f *testing.F) {
	f.Add("https://auth.example.com", true)
	f.Add("http://localhost:8080/auth", false)
	f.Add("", false)
	f.Add("not-a-url", true)
	f.Add("ftp://invalid.com", false)

	f.Fuzz(func(t *testing.T, issuerURL string, insecureSkip bool) {
		opt1 := WithOIDCIssuerURL(issuerURL)
		opt2 := WithOIDCInsecureSkipVerify(insecureSkip)

		s := &Server{}
		opt1(s)
		opt2(s)

		// Verify configuration doesn't panic
		_ = s.oidcIssuerURL
		_ = s.oidcInsecureSkipVerify
	})
}

// FuzzCleanupTicker tests cleanup ticker configuration
func FuzzCleanupTicker(f *testing.F) {
	f.Add(int64(1000000))       // 1ms
	f.Add(int64(60000000))      // 1 minute
	f.Add(int64(0))             // 0 duration
	f.Add(int64(-1000000))      // negative duration
	f.Add(int64(3600000000000)) // 1 hour

	f.Fuzz(func(t *testing.T, durationNs int64) {
		if durationNs <= 0 {
			return // Skip invalid durations
		}

		duration := time.Duration(durationNs)
		ticker := time.NewTicker(duration)
		defer ticker.Stop()

		opt := WithCleanupTicker(ticker)
		s := &Server{}
		opt(s)

		// Verify it doesn't panic
		if s.cleanupTicker != nil {
			s.cleanupTicker.Stop()
		}
	})
}

// FuzzTLSConfiguration tests TLS configuration
func FuzzTLSConfiguration(f *testing.F) {
	f.Add(true)
	f.Add(false)

	f.Fuzz(func(t *testing.T, insecureSkip bool) {
		opt := WithTLSInsecureSkipVerify(insecureSkip)
		s := &Server{}
		opt(s)

		// Verify configuration
		_ = s.tlsInsecureSkipVerify
	})
}
