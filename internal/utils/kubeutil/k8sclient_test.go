// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package kubeutil

import (
	"sync"
	"testing"

	v1alpha1 "github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
	"github.com/open-edge-platform/cluster-connect-gateway/internal/utils/certutil"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetCerts(t *testing.T) {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	v1alpha1.AddToScheme(scheme)

	// Create a fake client with a ClusterConnect object and secrets
	cc := &v1alpha1.ClusterConnect{
		ObjectMeta: ctrl.ObjectMeta{
			Name: "test-tunnel",
		},
		Spec: v1alpha1.ClusterConnectSpec{
			ClientCertRef: &corev1.ObjectReference{
				Name:      "client-cert",
				Namespace: "default",
			},
			ServerCertRef: &corev1.ObjectReference{
				Name:      "server-cert",
				Namespace: "default",
			},
		},
	}

	clientCertData, clientKeyData, err := certutil.GenerateTestCertificate()
	if err != nil {
		t.Fatalf("Failed to generate client certificate: %v", err)
	}

	clientCertSecret := &corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "client-cert",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"tls.crt": clientCertData,
			"tls.key": clientKeyData,
		},
	}

	serverCertData, serverKeyData, err := certutil.GenerateTestCertificate()
	if err != nil {
		t.Fatalf("Failed to generate server certificate: %v", err)
	}

	serverCertSecret := &corev1.Secret{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "server-cert",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"tls.crt": serverCertData,
			"tls.key": serverKeyData,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cc, clientCertSecret, serverCertSecret).Build()

	kc := &kubeclient{
		certStore: sync.Map{},
		kcStore:   sync.Map{},
		client:    fakeClient,
	}

	caPool, clientCert, err := kc.GetCerts("test-tunnel")
	assert.NoError(t, err)
	assert.NotNil(t, caPool)
	assert.NotNil(t, clientCert)
}

func TestInvalidateCerts(t *testing.T) {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	v1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	kc := &kubeclient{
		certStore: sync.Map{},
		client:    fakeClient,
	}

	// Store a dummy cert in the cert store
	kc.certStore.Store("test-tunnel", &Certs{})

	err := kc.InvalidateCerts("test-tunnel")
	assert.NoError(t, err)

	_, ok := kc.certStore.Load("test-tunnel")
	assert.False(t, ok)
}
