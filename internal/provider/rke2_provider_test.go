// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	rke2v1beta1 "github.com/rancher/cluster-api-provider-rke2/controlplane/api/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRKE2InjectFileFunc(t *testing.T) {
	tests := []struct {
		name     string
		obj      *unstructured.Unstructured
		filename string
		data     []byte
		wantErr  bool
	}{
		{
			name: "valid RKE2ControlPlane object",
			obj: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "controlplane.cluster.x-k8s.io/v1beta1",
					"kind":       "RKE2ControlPlane",
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
			err := RKE2InjectStaticPodManifestFunc(tt.obj, tt.filename, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("RKE2InjectFileFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				cp := &rke2v1beta1.RKE2ControlPlane{}
				if err := runtime.DefaultUnstructuredConverter.FromUnstructured(tt.obj.Object, cp); err != nil {
					t.Errorf("failed to convert control plane object to RKE2ControlPlane")
					return
				}

				if len(cp.Spec.Files) != 2 {
					t.Errorf("expected 2 file, got %d", len(cp.Spec.Files))
					return
				}

				file := cp.Spec.Files[1]
				expected := rke2StaticPodPath + tt.filename
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
