// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"

	k3sbootstrapv1beta2 "github.com/k3s-io/cluster-api-k3s/bootstrap/api/v1beta2"
	k3sv1beta2 "github.com/k3s-io/cluster-api-k3s/controlplane/api/v1beta2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const k3sStaticPodPath = "/var/lib/rancher/k3s/agent/pod-manifests/"

func K3SInjectStaticPodManifestFunc(obj *unstructured.Unstructured, filename string, data []byte) error {
	// Convert the given unstructured object to KThreesControlPlane.
	cp := &k3sv1beta2.KThreesControlPlane{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, cp)
	if err != nil || cp.Kind != "KThreesControlPlane" {
		return fmt.Errorf("failed to convert object to KthreesControlPlane")
	}

	// Check if the file already exists in the KThreesControlPlane spec.
	for _, file := range cp.Spec.KThreesConfigSpec.Files {
		if file.Path == k3sStaticPodPath+filename {
			return nil
		}
	}

	// Inject the file into the KThreesControlPlane spec.
	cp.Spec.KThreesConfigSpec.Files = append(cp.Spec.KThreesConfigSpec.Files, k3sbootstrapv1beta2.File{
		Path:    k3sStaticPodPath + filename,
		Content: string(data),
		Owner:   "root:root",
	})

	// Convert the KThreesControlPlane back to unstructured.
	updatedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cp)
	if err != nil {
		return fmt.Errorf("failed to convert KThreesControlPlane back to unstructured: %v", err)
	}

	// Update the object with the injected file.
	obj.Object = updatedObj
	return nil
}
