// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("JwtAuthorization", func() {
	var (
		jwtAuth *JwtAuthorization
	)

	const (
		projectId = "d60b7a96-6e85-457b-a0af-dead9234074e"
		tunnelId  = projectId + "-clustername-abcdef"
	)

	BeforeEach(func() {
		jwtAuth = &JwtAuthorization{
			JwtAuthenticator: &mockJwtAuthenticator{},
		}
	})

	Describe("extractTunnelId", func() {
		It("should extract tunnel ID from a valid path", func() {
			req := httptest.NewRequest(http.MethodGet, "/kubernetes/"+tunnelId, nil)
			id, err := extractTunnelId(req)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(tunnelId))
		})

		It("should return an error for an invalid path", func() {
			req := httptest.NewRequest(http.MethodGet, "/invalid/path", nil)
			_, err := extractTunnelId(req)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("extractProjectIdFromTunnel", func() {
		It("should extract project ID from a valid tunnel ID", func() {
			id, err := extractProjectIdFromTunnel(tunnelId)
			Expect(err).ToNot(HaveOccurred())
			Expect(id).To(Equal(projectId))
		})

		It("should return an error for an invalid tunnel ID", func() {
			_, err := extractProjectIdFromTunnel("invalidtunnelid")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("AuthMiddleware", func() {
		It("should return unauthorized for invalid token", func() {
			req := httptest.NewRequest(http.MethodGet, "/kubernetes/"+tunnelId, nil)
			req.Header.Set("Authorization", "Bearer invalid-token")
			rr := httptest.NewRecorder()

			handler := jwtAuth.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})

// mockJwtAuthenticator is a mock implementation of the JwtAuthenticator interface
type mockJwtAuthenticator struct{}

func (m *mockJwtAuthenticator) ParseAndValidate(token string) (jwt.Claims, error) {
	if token == "valid-token" {
		return jwt.MapClaims{"sub": "1234567890"}, nil
	}
	return nil, errors.New("invalid token")
}
func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}
