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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonv1alpha1 "sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/apis/v1alpha1"
)

// K8sAPIServerSpec defines the desired state of K8sAPIServer.
type K8sAPIServerSpec struct {
	// NestedComponentSpec contains the common and user-specified information that are
	// required for creating the component.
	// +optional
	NestedComponentSpec `json:",inline"`
}

// K8sAPIServerStatus defines the observed state of K8sAPIServer.
type K8sAPIServerStatus struct {
	// APIServerService is the reference to the service that expose the APIServer.
	// +optional
	APIServerService *corev1.ObjectReference `json:"apiserverService,omitempty"`

	// CommonStatus allows addons status monitoring.
	addonv1alpha1.CommonStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Namespaced,shortName=nkas,categories=capi;capn
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:subresource:status
// +kubebuilder:storageversion

// K8sAPIServer is the Schema for the nestedapiservers API.
type K8sAPIServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sAPIServerSpec   `json:"spec,omitempty"`
	Status K8sAPIServerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sAPIServerList contains a list of K8sAPIServer.
type K8sAPIServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sAPIServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sAPIServer{}, &K8sAPIServerList{})
}

var _ addonv1alpha1.CommonObject = &K8sAPIServer{}
var _ addonv1alpha1.Patchable = &K8sAPIServer{}

// ComponentName returns the name of the component for use with
// addonv1alpha1.CommonObject.
func (c *K8sAPIServer) ComponentName() string {
	return string(APIServer)
}

// CommonSpec returns the addons spec of the object allowing common funcs like
// Channel & Version to be usable.
func (c *K8sAPIServer) CommonSpec() addonv1alpha1.CommonSpec {
	return c.Spec.CommonSpec
}

// GetCommonStatus will return the common status for checking is a component
// was successfully deployed.
func (c *K8sAPIServer) GetCommonStatus() addonv1alpha1.CommonStatus {
	return c.Status.CommonStatus
}

// SetCommonStatus will set the status so that abstract representations can set
// Ready and Phases.
func (c *K8sAPIServer) SetCommonStatus(s addonv1alpha1.CommonStatus) {
	c.Status.CommonStatus = s
}

// PatchSpec returns the patches to be applied.
func (c *K8sAPIServer) PatchSpec() addonv1alpha1.PatchSpec {
	return c.Spec.PatchSpec
}
