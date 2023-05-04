package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows ReconcileDockerMachine to clean up resources associated with AWSMachine before
	// removing it from the apiserver.
	MachineFinalizer = "k8smachine.infrastructure.cluster.x-k8s.io"
)

// K8sMachineSpec defines the desired state of K8sMachine
type K8sMachineSpec struct {
	// ProviderID will be the container name in ProviderID format (docker:////<containername>)
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// CustomImage allows customizing the container image that is used for
	// running the machine
	// +optional
	CustomImage string `json:"customImage,omitempty"`
}

// K8sMachineStatus defines the observed state of K8sMachine
type K8sMachineStatus struct {
	// Ready denotes that the machine (docker container) is ready
	Ready bool `json:"ready"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the HCloudMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// GetConditions returns the observations of the operational state of the HCloudMachine resource.
func (r *K8sMachine) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions sets the underlying service state of the HCloudMachine to the predescribed clusterv1.Conditions.
func (r *K8sMachine) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

// +kubebuilder:resource:path=k8smachines,scope=Namespaced,categories=cluster-api
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status

// K8sMachine is the Schema for the k8smachines API
type K8sMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   K8sMachineSpec   `json:"spec,omitempty"`
	Status K8sMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// K8sMachineList contains a list of DockerMachine
type K8sMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []K8sMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&K8sMachine{}, &K8sMachineList{})
}
