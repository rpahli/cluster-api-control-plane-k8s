/*
Copyright 2020 The Kubernetes Authors.

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
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// K8sControlPlaneFinalizer is added to the K8sControlPlane to allow
	// nested deletions to happen before the object is cleaned up.
	K8sControlPlaneFinalizer = "k8s.controlplane.cluster.x-k8s.io"
)

// K8sControlPlaneSpec defines the desired state of K8sControlPlane.
type K8sControlPlaneSpec struct {
	// EtcdRef is the reference to the NestedEtcd.
	EtcdRef *corev1.ObjectReference `json:"etcd,omitempty"`

	// APIServerRef is the reference to the K8sAPIServer.
	// +optional
	APIServerRef *corev1.ObjectReference `json:"apiserver,omitempty"`

	// ContollerManagerRef is the reference to the NestedControllerManager.
	// +optional
	ControllerManagerRef *corev1.ObjectReference `json:"controllerManager,omitempty"`
}

// K8sControlPlaneStatus defines the observed state of K8sControlPlane.
type K8sControlPlaneStatus struct {
	// Etcd stores the connection information from the downstream etcd
	// implementation if the NestedEtcd type isn't used this
	// allows other component controllers to fetch the endpoints.
	// +optional
	Etcd *K8sControlPlaneStatusEtcd `json:"etcd,omitempty"`

	// APIServer stores the connection information from the control plane
	// this should contain anything shared between control plane components.
	// +optional
	APIServer *K8sControlPlaneStatusAPIServer `json:"apiserver,omitempty"`

	// Initialized denotes whether or not the control plane finished initializing.
	// +optional
	Initialized bool `json:"initialized"`

	// Ready denotes that the K8sControlPlane API Server is ready to
	// receive requests.
	// +kubebuilder:default=false
	Ready bool `json:"ready"`

	// ErrorMessage indicates that there is a terminal problem reconciling the
	// state, and will be set to a descriptive error message.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions specifies the conditions for the managed control plane
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// K8sControlPlaneStatusEtcd defines the status of the etcd component to
// allow other component controllers to take over the deployment.
type K8sControlPlaneStatusEtcd struct {
	// Addresses defines how to address the etcd instance
	Addresses []NestedEtcdAddress `json:"addresses,omitempty"`
}

// K8sControlPlaneStatusAPIServer defines the status of the APIServer
// component, this allows the next set of component controllers to take over
// the deployment.
type K8sControlPlaneStatusAPIServer struct {
	// ServiceCIDRs which is provided to kube-apiserver and kube-controller-manager.
	// +optional
	ServiceCIDR string `json:"serviceCidr,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Namespaced,shortName=ncp,categories=capi;capn
//+kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:subresource:status
// +k8s:defaulter-gen=true
// +kubebuilder:storageversion

// K8sControlPlane is the Schema for the k8scontrolplanes API.
type K8sControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sControlPlaneSpec   `json:"spec,omitempty"`
	Status K8sControlPlaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// K8sControlPlaneList contains a list of K8sControlPlane.
type K8sControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sControlPlane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sControlPlane{}, &K8sControlPlaneList{})
}

// GetOwnerCluster is a utility to return the owning clusterv1.Cluster.
func (r *K8sControlPlane) GetOwnerCluster(ctx context.Context, cli client.Client) (cluster *clusterv1.Cluster, err error) {
	return util.GetOwnerCluster(ctx, cli, r.ObjectMeta)
}

// GetConditions will return the conditions from the status.
func (r *K8sControlPlane) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions will reset the conditions to the new ones.
func (r *K8sControlPlane) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}
