// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0
package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-edge-platform/cluster-connect-gateway/internal/auth"
)

func TestGenerateToken(t *testing.T) {
	size := 8 // Example size for testing
	token, err := auth.GenerateToken(size)
	assert.NoError(t, err)
	assert.Equal(t, size*2, len(token)) // Token length in hex encoding
}
