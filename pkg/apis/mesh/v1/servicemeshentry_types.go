package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServiceMeshEntrySpec defines the desired state of ServiceMeshEntry
type ServiceMeshEntrySpec struct {
	// Must be formatted to conform to the DNS1123 specification.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// A list describes the properties of all ports of this service.
	// The Dubbo service port registered with MOSN is 20882,
	// otherwize the native Dubbo service port is 20880.
	// +kubebuilder:validation:MinItems=1
	Ports []*Port `json:"ports,omitempty"`

	// A list describes all registered instances of this service.
	Instances []*Instance `json:"instances,omitempty"`

	// Traffic policies of service-level
	Policy *Policy `json:"policy,omitempty"`

	// Subsets defined all sebsets of the current service.
	Subsets []*Subset `json:"subsets,omitempty"`

	// The Generation of MeshConfig, which to reconcile AppMeshConfig when MeshConfig changes.
	MeshConfigGeneration int64 `json:"meshConfigGeneration"`
}

// ServiceMeshEntryStatus defines the observed state of ServiceMeshEntry
type ServiceMeshEntryStatus struct {
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
	Phase          ConfigPhase  `json:"phase"`
	Status         *Status      `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceMeshEntry is the Schema for the servicemeshentries API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=servicemeshentries,scope=Namespaced
type ServiceMeshEntry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceMeshEntrySpec   `json:"spec,omitempty"`
	Status ServiceMeshEntryStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServiceMeshEntryList contains a list of ServiceMeshEntry
type ServiceMeshEntryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceMeshEntry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceMeshEntry{}, &ServiceMeshEntryList{})
}
