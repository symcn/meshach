package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceAccessorSpec defines the desired state of ServiceAccessor
type ServiceAccessorSpec struct {
	AccessHosts []string `json:"accessHosts"`
}

// ServiceAccessorStatus defines the observed state of ServiceAccessor
type ServiceAccessorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccessor is the Schema for the serviceaccessors API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=serviceaccessors,scope=Namespaced
type ServiceAccessor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceAccessorSpec   `json:"spec,omitempty"`
	Status ServiceAccessorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceAccessorList contains a list of ServiceAccessor
type ServiceAccessorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceAccessor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceAccessor{}, &ServiceAccessorList{})
}