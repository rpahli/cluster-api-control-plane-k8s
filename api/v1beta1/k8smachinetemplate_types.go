package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// K8sMachineTemplateSpec defines the desired state of K8sMachineTemplate
type K8sMachineTemplateSpec struct {
	Template K8sMachineTemplateResource `json:"template"`
}

// K8sMachineTemplateStatus defines the observed state of K8sMachineTemplate.
type K8sMachineTemplateStatus struct {
	// Capacity defines the resource capacity for this machine.
	// This value is used for autoscaling from zero operations as defined in:
	// https://github.com/kubernetes-sigs/cluster-api/blob/main/docs/proposals/20210310-opt-in-autoscaling-from-zero.md
	// +optional
	Capacity corev1.ResourceList `json:"capacity,omitempty"`

	// Conditions defines current service state of the HCloudMachineTemplate.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=k8smachinetemplates,scope=Namespaced,categories=cluster-api
// +kubebuilder:storageversion
// +kubebuilder:subresource:status

// K8sMachineTemplate is the Schema for the k8smachinetemplates API
type K8sMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sMachineTemplateSpec   `json:"spec,omitempty"`
	Status K8sMachineTemplateStatus `json:"status,omitempty"`
}

// GetConditions returns the observations of the operational state of the HCloudMachine resource.
func (r *K8sMachineTemplate) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions sets the underlying service state of the HCloudMachine to the predescribed clusterv1.Conditions.
func (r *K8sMachineTemplate) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

// +kubebuilder:object:root=true

// K8sMachineTemplateList contains a list of K8sMachineTemplate
type K8sMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sMachineTemplate{}, &K8sMachineTemplateList{})
}

// K8sMachineTemplateResource describes the data needed to create a DockerMachine from a template
type K8sMachineTemplateResource struct {
	// Spec is the specification of the desired behavior of the machine.
	Spec K8sMachineSpec `json:"spec"`
}
