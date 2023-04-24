//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1beta1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	apiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControlPlane) DeepCopyInto(out *K8sControlPlane) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControlPlane.
func (in *K8sControlPlane) DeepCopy() *K8sControlPlane {
	if in == nil {
		return nil
	}
	out := new(K8sControlPlane)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sControlPlane) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControlPlaneList) DeepCopyInto(out *K8sControlPlaneList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]K8sControlPlane, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControlPlaneList.
func (in *K8sControlPlaneList) DeepCopy() *K8sControlPlaneList {
	if in == nil {
		return nil
	}
	out := new(K8sControlPlaneList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sControlPlaneList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControlPlaneSpec) DeepCopyInto(out *K8sControlPlaneSpec) {
	*out = *in
	if in.EtcdRef != nil {
		in, out := &in.EtcdRef, &out.EtcdRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
	if in.APIServerRef != nil {
		in, out := &in.APIServerRef, &out.APIServerRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
	if in.ControllerManagerRef != nil {
		in, out := &in.ControllerManagerRef, &out.ControllerManagerRef
		*out = new(v1.ObjectReference)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControlPlaneSpec.
func (in *K8sControlPlaneSpec) DeepCopy() *K8sControlPlaneSpec {
	if in == nil {
		return nil
	}
	out := new(K8sControlPlaneSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControlPlaneStatus) DeepCopyInto(out *K8sControlPlaneStatus) {
	*out = *in
	if in.Etcd != nil {
		in, out := &in.Etcd, &out.Etcd
		*out = new(K8sControlPlaneStatusEtcd)
		(*in).DeepCopyInto(*out)
	}
	if in.APIServer != nil {
		in, out := &in.APIServer, &out.APIServer
		*out = new(K8sControlPlaneStatusAPIServer)
		**out = **in
	}
	if in.FailureMessage != nil {
		in, out := &in.FailureMessage, &out.FailureMessage
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(apiv1beta1.Conditions, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControlPlaneStatus.
func (in *K8sControlPlaneStatus) DeepCopy() *K8sControlPlaneStatus {
	if in == nil {
		return nil
	}
	out := new(K8sControlPlaneStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControlPlaneStatusAPIServer) DeepCopyInto(out *K8sControlPlaneStatusAPIServer) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControlPlaneStatusAPIServer.
func (in *K8sControlPlaneStatusAPIServer) DeepCopy() *K8sControlPlaneStatusAPIServer {
	if in == nil {
		return nil
	}
	out := new(K8sControlPlaneStatusAPIServer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControlPlaneStatusEtcd) DeepCopyInto(out *K8sControlPlaneStatusEtcd) {
	*out = *in
	if in.Addresses != nil {
		in, out := &in.Addresses, &out.Addresses
		*out = make([]NestedEtcdAddress, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControlPlaneStatusEtcd.
func (in *K8sControlPlaneStatusEtcd) DeepCopy() *K8sControlPlaneStatusEtcd {
	if in == nil {
		return nil
	}
	out := new(K8sControlPlaneStatusEtcd)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sAPIServer) DeepCopyInto(out *K8sAPIServer) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sAPIServer.
func (in *K8sAPIServer) DeepCopy() *K8sAPIServer {
	if in == nil {
		return nil
	}
	out := new(K8sAPIServer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sAPIServer) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sAPIServerList) DeepCopyInto(out *K8sAPIServerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]K8sAPIServer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sAPIServerList.
func (in *K8sAPIServerList) DeepCopy() *K8sAPIServerList {
	if in == nil {
		return nil
	}
	out := new(K8sAPIServerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sAPIServerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sAPIServerSpec) DeepCopyInto(out *K8sAPIServerSpec) {
	*out = *in
	in.NestedComponentSpec.DeepCopyInto(&out.NestedComponentSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sAPIServerSpec.
func (in *K8sAPIServerSpec) DeepCopy() *K8sAPIServerSpec {
	if in == nil {
		return nil
	}
	out := new(K8sAPIServerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sAPIServerStatus) DeepCopyInto(out *K8sAPIServerStatus) {
	*out = *in
	if in.APIServerService != nil {
		in, out := &in.APIServerService, &out.APIServerService
		*out = new(v1.ObjectReference)
		**out = **in
	}
	in.CommonStatus.DeepCopyInto(&out.CommonStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sAPIServerStatus.
func (in *K8sAPIServerStatus) DeepCopy() *K8sAPIServerStatus {
	if in == nil {
		return nil
	}
	out := new(K8sAPIServerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NestedComponentSpec) DeepCopyInto(out *NestedComponentSpec) {
	*out = *in
	out.CommonSpec = in.CommonSpec
	in.PatchSpec.DeepCopyInto(&out.PatchSpec)
	in.Resources.DeepCopyInto(&out.Resources)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NestedComponentSpec.
func (in *NestedComponentSpec) DeepCopy() *NestedComponentSpec {
	if in == nil {
		return nil
	}
	out := new(NestedComponentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControllerManager) DeepCopyInto(out *K8sControllerManager) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControllerManager.
func (in *K8sControllerManager) DeepCopy() *K8sControllerManager {
	if in == nil {
		return nil
	}
	out := new(K8sControllerManager)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sControllerManager) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControllerManagerList) DeepCopyInto(out *K8sControllerManagerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]K8sControllerManager, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControllerManagerList.
func (in *K8sControllerManagerList) DeepCopy() *K8sControllerManagerList {
	if in == nil {
		return nil
	}
	out := new(K8sControllerManagerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sControllerManagerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControllerManagerSpec) DeepCopyInto(out *K8sControllerManagerSpec) {
	*out = *in
	in.NestedComponentSpec.DeepCopyInto(&out.NestedComponentSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControllerManagerSpec.
func (in *K8sControllerManagerSpec) DeepCopy() *K8sControllerManagerSpec {
	if in == nil {
		return nil
	}
	out := new(K8sControllerManagerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sControllerManagerStatus) DeepCopyInto(out *K8sControllerManagerStatus) {
	*out = *in
	in.CommonStatus.DeepCopyInto(&out.CommonStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sControllerManagerStatus.
func (in *K8sControllerManagerStatus) DeepCopy() *K8sControllerManagerStatus {
	if in == nil {
		return nil
	}
	out := new(K8sControllerManagerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NestedEtcd) DeepCopyInto(out *NestedEtcd) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NestedEtcd.
func (in *NestedEtcd) DeepCopy() *NestedEtcd {
	if in == nil {
		return nil
	}
	out := new(NestedEtcd)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NestedEtcd) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NestedEtcdAddress) DeepCopyInto(out *NestedEtcdAddress) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NestedEtcdAddress.
func (in *NestedEtcdAddress) DeepCopy() *NestedEtcdAddress {
	if in == nil {
		return nil
	}
	out := new(NestedEtcdAddress)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NestedEtcdList) DeepCopyInto(out *NestedEtcdList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NestedEtcd, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NestedEtcdList.
func (in *NestedEtcdList) DeepCopy() *NestedEtcdList {
	if in == nil {
		return nil
	}
	out := new(NestedEtcdList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NestedEtcdList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NestedEtcdSpec) DeepCopyInto(out *NestedEtcdSpec) {
	*out = *in
	in.NestedComponentSpec.DeepCopyInto(&out.NestedComponentSpec)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NestedEtcdSpec.
func (in *NestedEtcdSpec) DeepCopy() *NestedEtcdSpec {
	if in == nil {
		return nil
	}
	out := new(NestedEtcdSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NestedEtcdStatus) DeepCopyInto(out *NestedEtcdStatus) {
	*out = *in
	if in.Addresses != nil {
		in, out := &in.Addresses, &out.Addresses
		*out = make([]NestedEtcdAddress, len(*in))
		copy(*out, *in)
	}
	in.CommonStatus.DeepCopyInto(&out.CommonStatus)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NestedEtcdStatus.
func (in *NestedEtcdStatus) DeepCopy() *NestedEtcdStatus {
	if in == nil {
		return nil
	}
	out := new(NestedEtcdStatus)
	in.DeepCopyInto(out)
	return out
}
