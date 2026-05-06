// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package certutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateTestCertificate generates a self-signed certificate and private key for testing.
// Returns PEM-encoded certificate and key byte slices.
func GenerateTestCertificate() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "kubernetes",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	privKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})
	return certPEM, keyPEM, nil
}

// GenerateTestJWT generates a valid JWT token for testing.
// Returns a Bearer token with HS256 signature.
func GenerateTestJWT() (string, error) {
	claims := jwt.MapClaims{
		"sub": "1234567890",
		"iat": time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte("test-signing-key"))
	if err != nil {
		return "", err
	}

	return "Bearer " + signedToken, nil
}
