// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

const testTunnelID = "test-tunnel-id"

func TestGetToken(t *testing.T) {
	scrtName := getTokenSecretName(testTunnelID)

	fakeClient := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      scrtName,
		},
		Data: map[string][]byte{
			"token": []byte("mockToken"),
		},
	})

	tokenManager := &manager{client: fakeClient.CoreV1().Secrets("test-ns"), namespace: "test-ns"}
	token, err := tokenManager.GetToken(context.TODO(), testTunnelID)
	if err != nil {
		t.Errorf("Failed to get token: %v", err)
	}

	assert.Equal(t, token.Value, "mockToken", "Token value mismatch")
}

func TestCreateAndStoreToken(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	tokenManager := &manager{client: fakeClient.CoreV1().Secrets("test-ns"), namespace: "test-ns"}

	cc := &v1alpha1.ClusterConnect{
		ObjectMeta: metav1.ObjectMeta{
			Name: testTunnelID,
			UID:  "fake-uid",
		},
	}

	err := tokenManager.CreateAndStoreToken(context.TODO(), testTunnelID, cc)
	if err != nil {
		t.Errorf("Failed to create and store token: %v", err)
	}

	secret, err := tokenManager.client.Get(context.TODO(), getTokenSecretName(testTunnelID), metav1.GetOptions{})
	if err != nil {
		t.Errorf("Failed to get token: %v", err)
	}

	assert.NotEmpty(t, secret.GetOwnerReferences())
	assert.Equal(t, string(secret.GetOwnerReferences()[0].UID), "fake-uid", "Token secret owner UID mismatch")
	assert.Equal(t, secret.GetOwnerReferences()[0].Name, testTunnelID, "Token secret owner name mismatch")
}

func TestTokenExist(t *testing.T) {
	scrtName := getTokenSecretName(testTunnelID)

	// Test the case that the token secret exists.
	fakeClientWithSecret := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      scrtName,
		},
		Data: map[string][]byte{
			"token": []byte("mockToken"),
		},
	})

	tokenManager := &manager{client: fakeClientWithSecret.CoreV1().Secrets("test-ns"), namespace: "test-ns"}
	exists, err := tokenManager.TokenExist(context.TODO(), testTunnelID)
	if err != nil {
		t.Errorf("Failed to check token existence: %v", err)
	}
	if !exists {
		t.Errorf("Token for tunnelID %s should exist but not found", testTunnelID)
	}

	// Test the case that the token does not exists.
	fakeClientWithoutSecret := fake.NewSimpleClientset()
	tokenManager = &manager{client: fakeClientWithoutSecret.CoreV1().Secrets("test-ns"), namespace: "test-ns"}
	exists, err = tokenManager.TokenExist(context.TODO(), testTunnelID)
	if err != nil {
		t.Errorf("Failed to check token existence: %v", err)
	}
	if exists {
		t.Errorf("Token for tunnelID %s should not exist but found", testTunnelID)
	}
}

func TestDeleteToken(t *testing.T) {
	scrtName := getTokenSecretName(testTunnelID)

	fakeClient := fake.NewSimpleClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      scrtName,
		},
		Data: map[string][]byte{
			"token": []byte("mockToken"),
		},
	})

	tokenManager := &manager{client: fakeClient.CoreV1().Secrets("test-ns"), namespace: "test-ns"}

	err := tokenManager.DeleteToken(context.TODO(), testTunnelID)
	if err != nil {
		t.Errorf("Failed to delete token: %v", err)
	}

	_, err = tokenManager.client.Get(context.TODO(), getTokenSecretName(testTunnelID), metav1.GetOptions{})
	if err == nil {
		t.Errorf("Get secret should fail but succeed")
	}
}
