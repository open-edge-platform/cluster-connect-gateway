// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"net/http"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/agent"
)

// MockAuthenticator is a mock implementation of the Authenticator interface
type MockAuthenticator struct {
	ParseAndValidateFunc func(tokenString string) (jwt.Claims, error)
}

func (m *MockAuthenticator) ParseAndValidate(tokenString string) (jwt.Claims, error) {
	return m.ParseAndValidateFunc(tokenString)
}

var _ = ginkgo.Describe("Auth Package", func() {
	var (
		req           *http.Request
		jwtAuthorizer JwtTokenAuthorizer
		mockAuth      *MockAuthenticator
	)

	ginkgo.BeforeEach(func() {
		req = &http.Request{
			Header: make(http.Header),
		}
		mockAuth = &MockAuthenticator{}
		jwtAuthorizer = JwtTokenAuthorizer{
			JwtAuth: mockAuth,
		}
	})

	ginkgo.Describe("extractTunnelId", func() {
		ginkgo.It("should return an error if the TunnelIdHeader is empty", func() {
			id, err := extractTunnelId(req)
			gomega.Expect(id).To(gomega.BeEmpty())
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("should return the tunnel ID if the TunnelIdHeader is present", func() {
			expectedID := "test-tunnel-id"
			req.Header.Set(agent.TunnelIdHeader, expectedID)
			id, err := extractTunnelId(req)
			gomega.Expect(id).To(gomega.Equal(expectedID))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

	ginkgo.Describe("JwtTokenAuthorizer.Authorizer", func() {
		ginkgo.It("should return an error if the TunnelIdHeader is empty", func() {
			clientKey, authed, err := jwtAuthorizer.Authorizer(req)
			gomega.Expect(clientKey).To(gomega.BeEmpty())
			gomega.Expect(authed).To(gomega.BeFalse())
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("should return an error if the token is invalid", func() {
			req.Header.Set(agent.TunnelIdHeader, "test-tunnel-id")
			req.Header.Set("Authorization", "Bearer invalid-token")

			mockAuth.ParseAndValidateFunc = func(tokenString string) (jwt.Claims, error) {
				return nil, errors.New("invalid token")
			}

			clientKey, authed, err := jwtAuthorizer.Authorizer(req)
			gomega.Expect(clientKey).To(gomega.Equal("test-tunnel-id"))
			gomega.Expect(authed).To(gomega.BeFalse())
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("should authorize a valid token", func() {
			req.Header.Set(agent.TunnelIdHeader, "test-tunnel-id")
			req.Header.Set("Authorization", "Bearer valid-token")

			mockAuth.ParseAndValidateFunc = func(tokenString string) (jwt.Claims, error) {
				return jwt.MapClaims{"foo": "bar"}, nil
			}

			clientKey, authed, err := jwtAuthorizer.Authorizer(req)
			gomega.Expect(clientKey).To(gomega.Equal("test-tunnel-id"))
			gomega.Expect(authed).To(gomega.BeTrue())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})

})

func TestAuth(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Auth Suite")
}
