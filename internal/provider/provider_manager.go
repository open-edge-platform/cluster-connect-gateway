// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

// ProviderManager is an interface that defines methods for managing CAPI control plane providers.
//
//go:generate mockery --name ProviderManager --filename provider_manager_mock.go --structname MockProviderManager --output ./mocks
type ProviderManager interface {
	// Register adds a static pod manifest path for a specific kind to the manager.
	Register(kind string, path string)
	StaticPodManifestPath(kind string) string
}

// ProviderManagerBuilder is a builder for ProviderManager.
type ProviderManagerBuilder struct {
	manager *manager
}

// NewProviderManagerBuilder creates a new builder for ProviderManager.
func NewProviderManager() *ProviderManagerBuilder {
	return &ProviderManagerBuilder{
		manager: &manager{
			pathMap: make(map[string]string),
		},
	}
}

// WithInjectStaticPodManifest adds an static pod manifest path for a specific kind to the manager.
func (b *ProviderManagerBuilder) WithProvider(kind string, path string) *ProviderManagerBuilder {
	b.manager.pathMap[kind] = path
	return b
}

// Build returns the constructed ProviderManager.
func (b *ProviderManagerBuilder) Build() ProviderManager {
	return b.manager
}

type manager struct {
	pathMap map[string]string
}

func (m *manager) Register(kind string, path string) {
	m.pathMap[kind] = path
}

func (m *manager) StaticPodManifestPath(kind string) string {
	path, ok := m.pathMap[kind]
	if !ok {
		return ""
	}

	return path
}
