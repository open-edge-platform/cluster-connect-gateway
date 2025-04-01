// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/agent"
)

func extractTunnelId(req *http.Request) (string, error) {
	id := req.Header.Get(agent.TunnelIdHeader)
	if id == "" {
		return id, fmt.Errorf("Empty header with tunnel id")
	}
	return id, nil
}

type Authenticator interface {
	ParseAndValidate(tokenString string) (jwt.Claims, error)
}

type JwtTokenAuthorizer struct {
	JwtAuth Authenticator
}

func (j *JwtTokenAuthorizer) Authorizer(req *http.Request) (clientKey string, authed bool, err error) {
	id, err := extractTunnelId(req)
	if err != nil {
		return id, false, err
	}

	token, err := request.BearerExtractor{}.ExtractToken(req)
	if err != nil {
		return id, false, err
	}

	_, err = j.JwtAuth.ParseAndValidate(token)

	if err != nil {
		return id, false, err
	}

	return id, true, nil
}

type SecretTokenAuthorizer struct {
	TokenManager TokenManager
}

func (s *SecretTokenAuthorizer) Authorizer(req *http.Request) (clientKey string, authed bool, err error) {
	id, err := extractTunnelId(req)
	if id == "" {
		return id, false, err
	}
	authToken := req.Header.Get(agent.TokenHeader)
	token, err := s.TokenManager.GetToken(req.Context(), id)
	if err != nil {
		return id, false, err
	}
	if token.Value == authToken {
		return id, true, nil
	}
	return id, false, nil
}
