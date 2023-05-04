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
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// NestedclusterImmutableMsg ...
const NestedclusterImmutableMsg = "Nestedclusters spec field is immutable. Please create a new resource instead. Ref doc: https://cluster-api.sigs.k8s.io/tasks/change-machine-template.html"

func (r *K8sCluster) SetupWebhookWithManager(mgr manager.Manager) error {
	return builder.WebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha4-nestedcluster,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=nestedclusters,versions=v1alpha4,name=validation.nestedclusters.infrastructure.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var _ webhook.Validator = &K8sCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *K8sCluster) ValidateCreate() error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *K8sCluster) ValidateUpdate(old runtime.Object) error {
	var allErrs field.ErrorList
	oldNestedcluster := old.(*K8sCluster)

	if !reflect.DeepEqual(r.Spec, oldNestedcluster.Spec) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "template", "spec"), r, NestedclusterImmutableMsg),
		)
	}

	if len(allErrs) != 0 {
		return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *K8sCluster) ValidateDelete() error {
	return nil
}
