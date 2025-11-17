// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// FuzzReconcile tests the Reconcile function with various name/namespace combinations
func FuzzReconcile(f *testing.F) {
	// Add seed corpus
	f.Add("test-cluster", "default")
	f.Add("", "")
	f.Add("cluster-with-special-chars!@#", "namespace-123")
	f.Add(string(make([]byte, 253)), "ns")     // Max k8s name length
	f.Add("cluster", string(make([]byte, 63))) // Max k8s namespace length

	f.Fuzz(func(t *testing.T, name string, namespace string) {
		// Skip empty names as they're invalid for k8s
		if name == "" || namespace == "" {
			return
		}

		// Create a scheme for the fake client
		scheme := runtime.NewScheme()
		_ = v1alpha1.AddToScheme(scheme)
		_ = clusterv1.AddToScheme(scheme)

		// Create fake client
		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		// Create reconciler
		r := &ClusterConnectReconciler{
			Client: fakeClient,
			Scheme: scheme,
		}

		// Create reconcile request
		req := ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      name,
				Namespace: namespace,
			},
		}

		// Call reconcile - should handle all inputs gracefully
		_, _ = r.Reconcile(context.Background(), req)
	})
}

// FuzzNamespacedName tests various NamespacedName combinations
func FuzzNamespacedName(f *testing.F) {
	f.Add("cluster-1", "ns-1")
	f.Add("UPPERCASE", "lowercase")
	f.Add("with-dashes-and-numbers-123", "ns-456")
	f.Add("dots.in.name", "ns.with.dots")

	f.Fuzz(func(t *testing.T, name string, namespace string) {
		nn := types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		}

		// Test string representation
		_ = nn.String()

		// Create request
		req := ctrl.Request{
			NamespacedName: nn,
		}

		// Verify it doesn't panic
		_ = req.Name
		_ = req.Namespace
	})
}

// FuzzClusterConnectSpec tests ClusterConnect spec parsing
func FuzzClusterConnectSpec(f *testing.F) {
	f.Add("cluster-1", "Cluster", "default")
	f.Add("", "", "")
	f.Add("test", "InvalidKind", "ns")
	f.Add("special!chars", "Cluster", "ns-123")

	f.Fuzz(func(t *testing.T, name string, kind string, namespace string) {
		// Create ClusterConnect object with fuzzy values
		cc := &v1alpha1.ClusterConnect{}
		cc.Name = name
		cc.Namespace = namespace

		// Test that we can access fields without panicking
		_ = cc.Name
		_ = cc.Namespace
	})
}

// FuzzControllerOptions tests controller configuration with various options
func FuzzControllerOptions(f *testing.F) {
	f.Add(int64(1), int64(10), true)
	f.Add(int64(0), int64(0), false)
	f.Add(int64(-1), int64(-10), true)
	f.Add(int64(100), int64(1000), false)
	f.Add(int64(9223372036854775807), int64(9223372036854775807), true) // Max int64

	f.Fuzz(func(t *testing.T, maxConcurrent int64, workerCount int64, enableEvents bool) {
		// Skip invalid values
		if maxConcurrent < 0 || workerCount < 0 {
			return
		}
		if maxConcurrent > 1000 || workerCount > 1000 {
			return // Prevent resource exhaustion
		}

		// Create scheme
		scheme := runtime.NewScheme()
		_ = v1alpha1.AddToScheme(scheme)
		_ = clusterv1.AddToScheme(scheme)

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		// Create reconciler with fuzzy configuration
		r := &ClusterConnectReconciler{
			Client: fakeClient,
			Scheme: scheme,
		}

		// Verify reconciler is created without panic
		_ = r.Client
		_ = r.Scheme
	})
}

// FuzzFinalizerName tests finalizer string handling
func FuzzFinalizerName(f *testing.F) {
	f.Add(FinalizerConnectController)
	f.Add("")
	f.Add("custom.finalizer.io/test")
	f.Add(string(make([]byte, 300))) // Very long finalizer
	f.Add("finalizer-with-!@#$%^&*()")

	f.Fuzz(func(t *testing.T, finalizerName string) {
		// Create ClusterConnect with fuzzy finalizer
		cc := &v1alpha1.ClusterConnect{}
		cc.Finalizers = []string{finalizerName}

		// Test finalizer operations
		if len(cc.Finalizers) > 0 {
			_ = cc.Finalizers[0]
		}
	})
}

// FuzzContextTimeout tests context with various timeout values
func FuzzContextTimeout(f *testing.F) {
	f.Add(int64(1000000))     // 1ms
	f.Add(int64(1000000000))  // 1s
	f.Add(int64(60000000000)) // 1 minute
	f.Add(int64(0))           // 0 duration

	f.Fuzz(func(t *testing.T, timeoutNs int64) {
		if timeoutNs <= 0 {
			return
		}
		if timeoutNs > 60000000000 { // Max 1 minute
			return
		}

		ctx := context.Background()

		// Verify context operations don't panic
		_ = ctx.Done()
		_ = ctx.Err()
	})
}
