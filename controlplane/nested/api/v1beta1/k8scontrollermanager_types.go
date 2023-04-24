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
	addonv1alpha1 "sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/apis/v1alpha1"
)

// K8sControllerManagerSpec defines the desired state of K8sControllerManager.
type K8sControllerManagerSpec struct {
	// NestedComponentSpec contains the common and user-specified information
	// that are required for creating the component.
	// +optional
	NestedComponentSpec `json:",inline"`
}

// K8sControllerManagerStatus defines the observed state of K8sControllerManager.
type K8sControllerManagerStatus struct {
	// CommonStatus allows addons status monitoring.
	addonv1alpha1.CommonStatus `json:",inline"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Namespaced,shortName=nkcm,categories=capi;capn
//+kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
//+kubebuilder:subresource:status
// +kubebuilder:storageversion

// K8sControllerManager is the Schema for the nestedcontrollermanagers API.
type K8sControllerManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sControllerManagerSpec   `json:"spec,omitempty"`
	Status K8sControllerManagerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// K8sControllerManagerList contains a list of K8sControllerManager.
type K8sControllerManagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sControllerManager `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sControllerManager{}, &K8sControllerManagerList{})
}

var _ addonv1alpha1.CommonObject = &K8sControllerManager{}
var _ addonv1alpha1.Patchable = &K8sControllerManager{}

// ComponentName returns the name of the component for use with
// addonv1alpha1.CommonObject.
func (c *K8sControllerManager) ComponentName() string {
	return string(ControllerManager)
}

// CommonSpec returns the addons spec of the object allowing common funcs like
// Channel & Version to be usable.
func (c *K8sControllerManager) CommonSpec() addonv1alpha1.CommonSpec {
	return c.Spec.CommonSpec
}

// GetCommonStatus will return the common status for checking is a component
// was successfully deployed.
func (c *K8sControllerManager) GetCommonStatus() addonv1alpha1.CommonStatus {
	return c.Status.CommonStatus
}

// SetCommonStatus will set the status so that abstract representations can set
// Ready and Phases.
func (c *K8sControllerManager) SetCommonStatus(s addonv1alpha1.CommonStatus) {
	c.Status.CommonStatus = s
}

// PatchSpec returns the patches to be applied.
func (c *K8sControllerManager) PatchSpec() addonv1alpha1.PatchSpec {
	return c.Spec.PatchSpec
}
