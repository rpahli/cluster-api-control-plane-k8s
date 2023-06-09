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

package cache

import (
	corev1 "k8s.io/api/core/v1"
)

type Cache interface {
	AddTenant(string)
	RemoveTenant(string) error
	GetNamespace(string) *Namespace
	AddNamespace(*Namespace) error
	RemoveNamespace(*Namespace) error
	UpdateNamespace(*Namespace, *Namespace) error
	AddCluster(*Cluster) error
	RemoveCluster(string) error
	GetPod(string) *Pod
	AddPod(*Pod) error
	RemovePod(*Pod) error
	AddProvision(string, string, []*Slice) error
	RemoveProvision(string, string) error
	UpdateClusterCapacity(string, corev1.ResourceList) error
	SnapshotForNamespaceSched(...*Namespace) (*NamespaceSchedSnapshot, error)
	SnapshotForPodSched(pod *Pod) (*PodSchedSnapshot, error)
	Dump() string
}
