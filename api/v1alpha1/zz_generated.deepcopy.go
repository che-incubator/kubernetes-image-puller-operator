// +build !ignore_autogenerated

//
// Copyright (c) 2012-2021 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesImagePuller) DeepCopyInto(out *KubernetesImagePuller) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesImagePuller.
func (in *KubernetesImagePuller) DeepCopy() *KubernetesImagePuller {
	if in == nil {
		return nil
	}
	out := new(KubernetesImagePuller)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubernetesImagePuller) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesImagePullerConfig) DeepCopyInto(out *KubernetesImagePullerConfig) {
	*out = *in
	if in.configMap != nil {
		in, out := &in.configMap, &out.configMap
		*out = new(v1.ConfigMap)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesImagePullerConfig.
func (in *KubernetesImagePullerConfig) DeepCopy() *KubernetesImagePullerConfig {
	if in == nil {
		return nil
	}
	out := new(KubernetesImagePullerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesImagePullerList) DeepCopyInto(out *KubernetesImagePullerList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]KubernetesImagePuller, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesImagePullerList.
func (in *KubernetesImagePullerList) DeepCopy() *KubernetesImagePullerList {
	if in == nil {
		return nil
	}
	out := new(KubernetesImagePullerList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *KubernetesImagePullerList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesImagePullerSpec) DeepCopyInto(out *KubernetesImagePullerSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesImagePullerSpec.
func (in *KubernetesImagePullerSpec) DeepCopy() *KubernetesImagePullerSpec {
	if in == nil {
		return nil
	}
	out := new(KubernetesImagePullerSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *KubernetesImagePullerStatus) DeepCopyInto(out *KubernetesImagePullerStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KubernetesImagePullerStatus.
func (in *KubernetesImagePullerStatus) DeepCopy() *KubernetesImagePullerStatus {
	if in == nil {
		return nil
	}
	out := new(KubernetesImagePullerStatus)
	in.DeepCopyInto(out)
	return out
}
