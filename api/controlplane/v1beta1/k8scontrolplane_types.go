package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
)

// K8sControlPlanSpec defines the desired state of K8sControlPlane.
type K8sControlPlanSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint            `json:"controlPlaneEndpoint"`
	Namespace            string                           `json:"namespace,omitempty"`
	Version              string                           `json:"version,omitempty"`
	ClusterConfiguration bootstrapv1.ClusterConfiguration `json:"clusterConfiguration,omitempty"`
}

// K8sControlPlanStatus defines the observed state of K8sControlPlane.
type K8sControlPlanStatus struct {
	// Ready is when the NestedControlPlane has a API server URL.
	// +optional
	Ready                       bool                 `json:"ready,omitempty"`
	Initialized                 bool                 `json:"initialized,omitempty"`
	Conditions                  clusterv1.Conditions `json:"conditions,omitempty"`
	ExternalManagedControlPlane bool                 `json:"externalManagedControlPlane,omitempty"`
	Version                     string               `json:"version,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=k8scontrolplanes,scope=Namespaced,shortName=nc,categories=capi;capn
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
//+kubebuilder:printcolumn:name="Initialized",type="boolean",JSONPath=".status.initialized"
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".metadata.spec.version"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:subresource:status
// +kubebuilder:storageversion

// K8sControlPlane is the Schema for the k8sclusters API.
type K8sControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sControlPlanSpec   `json:"spec,omitempty"`
	Status K8sControlPlanStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sControlPlaneList contains a list of K8sControlPlane.
type K8sControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sControlPlane `json:"items"`
}

// GetConditions returns the observations of the operational state of the HetznerCluster resource.
func (r *K8sControlPlane) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions sets the underlying service state of the HetznerCluster to the predescribed clusterv1.Conditions.
func (r *K8sControlPlane) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&K8sControlPlane{}, &K8sControlPlaneList{})
}
