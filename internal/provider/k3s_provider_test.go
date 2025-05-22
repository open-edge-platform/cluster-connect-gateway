// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	k3sv1beta2 "github.com/k3s-io/cluster-api-k3s/controlplane/api/v1beta2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestK3SInjectFileFunc(t *testing.T) {
	tests := []struct {
		name     string
		obj      *unstructured.Unstructured
		filename string
		data     []byte
		wantErr  bool
	}{
		{
			name: "valid K3SControlPlane object",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "controlplane.cluster.x-k8s.io/v1beta2",
					"kind":       "KThreesControlPlane",
					"metadata": map[string]interface{}{
						"name": "test-control-plane",
					},
					"spec": map[string]interface{}{
						"kthreesConfigSpec": map[string]interface{}{
							"files": []interface{}{
								map[string]interface{}{
									"path":    "/etc/existing/file/config.yaml",
									"content": "some existing file data",
									"owner":   "root:root",
								},
							},
						},
					},
				},
			},
			filename: "config.yaml",
			data:     []byte("some configuration data"),
			wantErr:  false,
		},
		{
			name: "invalid object type",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "controlplane.cluster.x-k8s.io/v1beta1",
					"kind":       "KubeadmControlPlane",
					"metadata": map[string]interface{}{
						"name": "test-control-plane",
					},
					"spec": map[string]interface{}{
						"files": []interface{}{
							map[string]interface{}{
								"path":    "/etc/existing/file/config.yaml",
								"content": "some existing file data",
								"owner":   "root:root",
							},
						},
					},
				},
			},
			filename: "config.yaml",
			data:     []byte("some configuration data"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := K3SInjectStaticPodManifestFunc(tt.obj, tt.filename, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("K3SInjectFileFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				cp := &k3sv1beta2.KThreesControlPlane{}
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(tt.obj.Object, cp); err != nil {
					t.Errorf("failed to convert control plane object to K3SControlPlane")
					return
				}

				if len(cp.Spec.KThreesConfigSpec.Files) != 2 {
					t.Errorf("expected 2 file, got %d", len(cp.Spec.KThreesConfigSpec.Files))
					return
				}

				file := cp.Spec.KThreesConfigSpec.Files[1]
				expected := k3sStaticPodPath + tt.filename
				if file.Path != expected {
					t.Errorf("expected path %s, got %s", expected, file.Path)
				}
				if file.Content != string(tt.data) {
					t.Errorf("expected content %s, got %s", string(tt.data), file.Content)
				}
				if file.Owner != "root:root" {
					t.Errorf("expected owner root:root, got %s", file.Owner)
				}
			}
		})
	}
}
