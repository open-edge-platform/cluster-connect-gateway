// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"

	rke2bootstrapv1beta1 "github.com/rancher/cluster-api-provider-rke2/bootstrap/api/v1beta1"
	rke2v1beta1 "github.com/rancher/cluster-api-provider-rke2/controlplane/api/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

const rke2StaticPodPath = "/var/lib/rancher/rke2/agent/pod-manifests/"

func RKE2InjectStaticPodManifestFunc(obj *unstructured.Unstructured, filename string, data []byte) error {
	// Convert the given unstructured object to RKE2ControlPlane.
	cp := &rke2v1beta1.RKE2ControlPlane{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, cp)
	if err != nil || cp.Kind != "RKE2ControlPlane" {
		return fmt.Errorf("failed to convert object to RKE2ControlPlane")
	}

	// Check if the file already exists in the RKE2ControlPlane spec.
	for _, file := range cp.Spec.Files {
		if file.Path == rke2StaticPodPath+filename {
			return nil
		}
	}

	// Inject the file into the RKE2ControlPlane spec.
	cp.Spec.Files = append(cp.Spec.Files, rke2bootstrapv1beta1.File{
		Path:    rke2StaticPodPath + filename,
		Content: string(data),
		Owner:   "root:root",
	})

	// Convert the RKE2ControlPlane back to unstructured.
	updatedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cp)
	if err != nil {
		return fmt.Errorf("failed to convert RKE2ControlPlane back to unstructured: %v", err)
	}

	// Update the object with the injected file.
	obj.Object = updatedObj
	return nil
}
