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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KubernetesImagePullerSpec defines the desired state of KubernetesImagePuller
type KubernetesImagePullerSpec struct {
	ConfigMapName        string `json:"configMapName,omitempty"`
	DaemonsetName        string `json:"daemonsetName,omitempty"`
	DeploymentName       string `json:"deploymentName,omitempty"`
	Images               string `json:"images,omitempty"`
	CachingIntervalHours string `json:"cachingIntervalHours,omitempty"`
	CachingMemoryRequest string `json:"cachingMemoryRequest,omitempty"`
	CachingMemoryLimit   string `json:"cachingMemoryLimit,omitempty"`
	CachingCpuRequest    string `json:"cachingCPURequest,omitempty"`
	CachingCpuLimit      string `json:"cachingCPULimit,omitempty"`
	NodeSelector         string `json:"nodeSelector,omitempty"`
	ImagePullSecrets     string `json:"imagePullSecrets,omitempty"`
	Affinity             string `json:"affinity,omitempty"`
	ImagePullerImage     string `json:"imagePullerImage,omitempty"`
}

// KubernetesImagePullerStatus defines the observed state of KubernetesImagePuller
type KubernetesImagePullerStatus struct {
	ImagePullerImage string `json:"imagePullerImage,omitempty"`
}

// KubernetesImagePuller is the Schema for the kubernetesimagepullers API
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:resource:path=kubernetesimagepullers,scope=Namespaced
type KubernetesImagePuller struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubernetesImagePullerSpec   `json:"spec,omitempty"`
	Status KubernetesImagePullerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KubernetesImagePullerList contains a list of KubernetesImagePuller
type KubernetesImagePullerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesImagePuller `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubernetesImagePuller{}, &KubernetesImagePullerList{})
}

type KubernetesImagePullerConfig struct {
	configMap *corev1.ConfigMap
}

func (config *KubernetesImagePullerConfig) WithDaemonsetName(name string) *KubernetesImagePullerConfig {
	config.configMap.Data["DAEMONSET_NAME"] = name
	return &KubernetesImagePullerConfig{
		configMap: config.configMap,
	}
}
