// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
)

const (
	DefaultSecretNamespace = "connect-gateway-secrets"
	DefaultTokenLength     = 54
)

var GetClusterConfig = rest.InClusterConfig

// NewInClusterSecretTokenManager creates a new TokenManager implementation
// that uses local Kubernetes Secrets as a store for the token.
func NewTokenManager() (TokenManager, error) {
	restconfig, err := GetClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to obtain in-cluster config %v", err)
	}

	clientset, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client %v", err)
	}

	namespace := DefaultSecretNamespace
	ns, ok := os.LookupEnv("SECRET_NAMESPACE")
	if ok && ns != "" {
		namespace = ns
	}

	return &manager{
		client:    clientset.CoreV1().Secrets(namespace),
		namespace: namespace,
	}, nil
}

// manager is the implementation of TokenManager interface
type manager struct {
	client    v1.SecretInterface
	namespace string

	// TODO: Implement cache
	// cache     sync.Map
}

// TokenExist returns true if the token secret for a given tunnel ID exists
func (m *manager) TokenExist(ctx context.Context, tunnelID string) (bool, error) {
	if _, err := m.client.Get(ctx, getTokenSecretName(tunnelID), metav1.GetOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

// GetTokenSecretWithTunnelID returns the token value for a given tunnel ID.
func (m *manager) GetToken(ctx context.Context, tunnelID string) (*Token, error) {
	// Find it from Cache. If not exists or expired, retrieve from K8s Secret.

	// Attempt to get secret
	secret, err := m.client.Get(ctx, getTokenSecretName(tunnelID), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get token for %s (%v)", tunnelID, err)
	}

	return &Token{Value: string(secret.Data["token"])}, nil
}

// CreateAndStoreToken generates a token value and create a Secret with it for a given tunnel ID.
func (m *manager) CreateAndStoreToken(ctx context.Context, tunnelID string, cc *v1alpha1.ClusterConnect) error {
	token, err := GenerateToken(DefaultTokenLength)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getTokenSecretName(tunnelID),
			Namespace: m.namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.ClusterConnectKind,
					Name:       cc.Name,
					UID:        cc.UID,
				},
			},
		},
		Data: map[string][]byte{
			"token": []byte(token),
			// TODO: Add expiration
		},
	}

	if _, err := m.client.Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create token secret (%v)", err)
		}
	}

	return nil
}

// Delete token removes the token secret for a given tunnel ID.
func (m *manager) DeleteToken(ctx context.Context, tunnelID string) error {
	if err := m.client.Delete(ctx, getTokenSecretName(tunnelID), metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete token secret (%v)", err)
	}
	return nil
}

// GetTokenSecretName returns the token secret name for a given ClusterConnect object.
func getTokenSecretName(tunnelId string) string {
	// TODO: need to think about a case where tunnel ID exceeds 247 characters
	// which makes the Secret name exceeds K8s resource name limit 253
	return tunnelId + "-agent-token"
}
