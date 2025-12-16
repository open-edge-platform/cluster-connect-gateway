// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	v1alpha1 "github.com/open-edge-platform/cluster-connect-gateway/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1beta2conditions "sigs.k8s.io/cluster-api/util/conditions/v1beta2"
)

// requiredConditionTypes lists the conditions that must be met for the
// ClusterConnect status to be considered Ready.
var requiredConditionTypes = []string{
	v1alpha1.AuthTokenReadyCondition,
	v1alpha1.AgentManifestGeneratedCondition,
	v1alpha1.ControlPlaneEndpointSetCondition,
	v1alpha1.ClusterSpecUpdatedCondition,
	v1alpha1.TopologyReconciledCondition,
	v1alpha1.ConnectionProbeCondition,
}

// initConditions initializes conditions with Unknown if conditions are not set.
func initConditions(cc *v1alpha1.ClusterConnect) {
	if len(cc.GetConditions()) < len(requiredConditionTypes) {
		for _, condition := range requiredConditionTypes {
			// Skip ClusterSpecUpdatedCondition and TopologyReconciledCondition if ClusterRef is not set.
			if cc.Spec.ClusterRef == nil {
				if condition == v1alpha1.ClusterSpecUpdatedCondition ||
					condition == v1alpha1.TopologyReconciledCondition {
					continue
				}
			}

			if v1beta2conditions.Has(cc, condition) {
				// If the condition is already set, skip it.
				continue
			}

			// Set condition to Unknown otherwise.
			v1beta2conditions.Set(cc, metav1.Condition{
				Type:   condition,
				Status: metav1.ConditionUnknown,
				Reason: v1alpha1.ReadyUnknownReason,
			})
		}
	}
}

func setAuthTokenReadyConditionTrue(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.AuthTokenReadyCondition,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ReadyReason,
		Message: conditionMessage,
	})
}

func setAuthTokenReadyConditionFalse(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.AuthTokenReadyCondition,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.NotReadyReason,
		Message: conditionMessage,
	})
}

func setAgentManifestGeneratedConditionTrue(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.AgentManifestGeneratedCondition,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ReadyReason,
		Message: conditionMessage,
	})
}

func setAgentManifestGeneratedConditionFalse(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.AgentManifestGeneratedCondition,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.NotReadyReason,
		Message: conditionMessage,
	})
}

func setControlPlaneEndpointSetConditionTrue(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.ControlPlaneEndpointSetCondition,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ReadyReason,
		Message: conditionMessage,
	})
}

func setClusterSpecReadyConditionTrue(cc *v1alpha1.ClusterConnect, message ...string) { //nolint:unparam
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.ClusterSpecUpdatedCondition,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ReadyReason,
		Message: conditionMessage,
	})
}

func setClusterSpecUpdatedConditionFalse(cc *v1alpha1.ClusterConnect, message ...string) { //nolint:unparam
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.ClusterSpecUpdatedCondition,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.NotReadyReason,
		Message: conditionMessage,
	})
}

func setTopologyReconciledConditionTrue(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.TopologyReconciledCondition,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ReadyReason,
		Message: conditionMessage,
	})
}

func setTopologyReconciledConditionFalse(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.TopologyReconciledCondition,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.NotReadyReason,
		Message: conditionMessage,
	})
}

func setKubeconfigReadyConditionTrue(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.KubeconfigReadyCondition,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ReadyReason,
		Message: conditionMessage,
	})
}

func setConnectionProbeConditionTrue(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.ConnectionProbeCondition,
		Status:  metav1.ConditionTrue,
		Reason:  v1alpha1.ConnectionProbeSucceededReason,
		Message: conditionMessage,
	})
}
func setConnectionProbeConditionFalse(cc *v1alpha1.ClusterConnect, message ...string) {
	conditionMessage := ""
	if len(message) > 0 {
		conditionMessage = message[0]
	}
	v1beta2conditions.Set(cc, metav1.Condition{
		Type:    v1alpha1.ConnectionProbeCondition,
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.ConnectionProbeFailedReason,
		Message: conditionMessage,
	})
}
