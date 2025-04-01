// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
)

// TokenManager interface defines methods for retrieving, storing, and deleting tokens.
//
//go:generate mockery --name TokenManager --filename token_manager_mock.go --structname MockTokenManager --output ./mocks
type TokenManager interface {
	GetToken(ctx context.Context, tunnelID string) (*Token, error)                                                          // GetToken retrieves token for a given tunnel ID.
	TokenExist(ctx context.Context, tunnelID string) (bool, error)                                                          // TokenExist returns true if the token for a given tunnel ID alreay exists.
	CreateAndStoreToken(ctx context.Context, tunnelID string /* , tokenTTLHours int, */, cc *v1alpha1.ClusterConnect) error // CreateToken creates and stores a token with its value and TTL in hours.
	DeleteToken(ctx context.Context, tunnelID string) error                                                                 // DeleteToken deletes a token for a given tunnel ID.
	// RefreshToken(ctx context.Context, tunnelID string, tokenTTLHours int) error // TODO: implement
}

// Token struct represents a token with its value, updated time, and time-to-live in hours.
type Token struct {
	Value string

	// TODO: Add expiration
	// Expire time.Time
}

// GenerateToken generates a random string to be used as a token for authenticating the connect-agent.
func GenerateToken(size int) (string, error) {
	token := make([]byte, size)

	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(token), err
}
