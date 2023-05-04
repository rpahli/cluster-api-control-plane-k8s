/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// K8sClusterSpec defines the desired state of K8sCluster.
type K8sClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`
	Namespace            string                `json:"namespace,omitempty"`
}

// K8sClusterStatus defines the observed state of K8sCluster.
type K8sClusterStatus struct {
	// Ready is when the NestedControlPlane has a API server URL.
	// +optional
	Ready bool `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:path=k8sclusters,scope=Namespaced,shortName=nc,categories=capi;capn
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:subresource:status
// +kubebuilder:storageversion

// K8sCluster is the Schema for the nestedclusters API.
type K8sCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sClusterSpec   `json:"spec,omitempty"`
	Status K8sClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sClusterList contains a list of K8sCluster.
type K8sClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sCluster{}, &K8sClusterList{})
}
