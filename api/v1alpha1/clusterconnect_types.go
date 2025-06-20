// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	ClusterConnectKind = "ClusterConnect"

	// AuthTokenReadyCondition reports if an authentication token Secret for an object is ready.
	AuthTokenReadyCondition = "AuthTokenReady"

	// ConnectAgentManifestReadyCondition reports if the agent pod manifest is ready in status.
	AgentManifestGeneratedCondition = "ConnectAgentManifestGenerated"

	// ControlPlaneEndpointSetCondition reports if the ControlPlane endpoint URL is generated.
	ControlPlaneEndpointSetCondition = "ControlPlaneEndpointSet"

	// ClusterSpecUpdated reports if the Cluster spec is updated with connect agent configuration.
	// Note: This condition is valid only when CAPI ClusterRef is set.
	ClusterSpecUpdatedCondition = "ClusterSpecUpdated"

	// TopologyReconciled reports if the ControlPlane spec is updated with agent pod manifest.
	// Note: This condition is valid only when CAPI ClusterRef is set.
	TopologyReconciledCondition = "TopologyReconciled"

	// KubeconfigReadyCondition reports if the kubeconfig Secret is ready.
	// Note: This condition is valid only when CAPI ClusterRef is set.
	KubeconfigReadyCondition = "KubeconfigReady"

	// ReadyReason applies to a condition surfacing object readiness.
	ReadyReason = "Ready"

	// NotReadyReason applies to a condition surfacing object not satisfying readiness criteria.
	NotReadyReason = "NotReady"

	// ReadyUnknownReason applies to a condition surfacing object readiness unknown.
	ReadyUnknownReason = "ReadyUnknown"
)

// ConnectionProbe condition and corresponding reasons.
const (
	ConnectionProbeCondition = "ConnectionProbe"

	// ConnectionProbeFailedReason surfaces issues with the connection to the workload's cluster connect-agent.
	ConnectionProbeFailedReason = "ProbeFailed"

	// ConnectionProbeSucceededReason is used to report a working connection with the workload's cluster connect-agent.
	ConnectionProbeSucceededReason = "ProbeSucceeded"
)

// ClusterConnectSpec defines the desired state of ClusterConnect.
type ClusterConnectSpec struct {
	// ClusterRef is an optional reference to a CAPI provider-specific resource that holds
	// the details for the Cluster to connect.
	// +optional
	ClusterRef *corev1.ObjectReference `json:"clusterRef,omitempty"`

	// ServerCertRef is an optional reference to a PEM-encoded server certificate authority data for the kubeapi-server to proxy.
	// The secret format is intended to match the format of the <cluster-name>-ca secret used in CAPI.
	// +optional
	ServerCertRef *corev1.ObjectReference `json:"serverCertRef,omitempty"`

	// ClientCertRef is an optional reference to a PEM-encoded client certificates for the cluster administrator
	// to authenticate with the kubeapi-server.
	// The secret format is intended to match the format of the <cluster-name>-cca secret used in cluster-api.
	// +optional
	ClientCertRef *corev1.ObjectReference `json:"clientCertRef,omitempty"`
}

// ClusterConnectStatus defines the observed state of ClusterConnect.
type ClusterConnectStatus struct {
	// Ready indicates connect-agent pod manifest is ready to be consumed.
	// +optional
	Ready bool `json:"ready,omitempty"`

	// ControlPlaneEndpoint provides the URL for accessing the kubeapi-server through the connection gateway.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`

	// AgentManifest is the connect-agent Pod manifest.
	// +optional
	AgentManifest string `json:"agentManifest,omitempty"`

	// ConnectionProbe defines the state of the connection with connect-agent.
	ConnectionProbe ConnectionProbeState `json:"connectionProbe,omitempty"`

	// Conditions defines current connection state of the cluster.
	// Known condition types are TBD.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type ConnectionProbeState struct {
	// LastProbeTimestamp is the time when the health probe was executed last.
	LastProbeTimestamp metav1.Time `json:"lastProbeTimestamp,omitempty"`

	// LastProbeSuccessTimestamp is the time when the health probe was successfully executed last.
	LastProbeSuccessTimestamp metav1.Time `json:"lastProbeSuccessTimestamp,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=clusterconnects,shortName=ccon,scope=Cluster
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of this resource"

// ClusterConnect is the Schema for the clusterconnects API.
type ClusterConnect struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterConnectSpec   `json:"spec,omitempty"`
	Status ClusterConnectStatus `json:"status,omitempty"`
}

// GetTunnelID returns the tunnel ID.
// ClusterConnect object ID is used as globally unique tunnel ID.
func (c *ClusterConnect) GetTunnelID() string {
	return c.Name
}

// GetConditions returns the set of conditions for this object.
func (c *ClusterConnect) GetConditions() []metav1.Condition {
	return c.Status.Conditions
}

// GetV1Beta2Conditions returns the set of conditions for this object.
// Implements Cluster API condition setter interface.
func (c *ClusterConnect) GetV1Beta2Conditions() []metav1.Condition {
	return c.Status.Conditions
}

// SetConditions sets the conditions on this object.
func (c *ClusterConnect) SetConditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

// SetV1Beta2Conditions sets the conditions on this object.
// Implements Cluster API condition setter interface.
func (c *ClusterConnect) SetV1Beta2Conditions(conditions []metav1.Condition) {
	c.Status.Conditions = conditions
}

// +kubebuilder:object:root=true

// ClusterConnectList contains a list of ClusterConnect.
type ClusterConnectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterConnect `json:"items"`
}

func init() {
	objectTypes = append(objectTypes, &ClusterConnect{}, &ClusterConnectList{})
}
