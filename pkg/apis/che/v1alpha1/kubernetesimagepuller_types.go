package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KubernetesImagePullerSpec defines the desired state of KubernetesImagePuller
type KubernetesImagePullerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// KubernetesImagePullerStatus defines the observed state of KubernetesImagePuller
type KubernetesImagePullerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubernetesImagePuller is the Schema for the kubernetesimagepullers API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=kubernetesimagepullers,scope=Namespaced
type KubernetesImagePuller struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubernetesImagePullerSpec   `json:"spec,omitempty"`
	Status KubernetesImagePullerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubernetesImagePullerList contains a list of KubernetesImagePuller
type KubernetesImagePullerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesImagePuller `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubernetesImagePuller{}, &KubernetesImagePullerList{})
}
