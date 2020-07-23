package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// StringMatchType ...
type StringMatchType string

// StringMatchType enum
var (
	Exact  StringMatchType = "exact"
	Prefix StringMatchType = "prefix"
	Regex  StringMatchType = "regex"
)

// MeshConfigSpec defines the desired state of MeshConfig
type MeshConfigSpec struct {
	MatchHeaderLabelKeys   map[string]StringMatchType `json:"matchHeaderLabelKeys"`
	MatchSourceLabelKeys   []string                   `json:"matchSourceLabelKeys"`
	WorkloadEntryLabelKeys []string                   `json:"workloadEntryLabelKeys"`
	MeshLabelsRemap        map[string]string          `json:"meshLabelsRemap"`
	ExtractedLabels        []string                   `json:"extractedLabels"`
	SidecarSelectLabel     string                     `json:"sidecarSelectLabel"`
	SidecarDefaultHosts    []string                   `json:"sidecarDefaultHosts"`
	GlobalSubsets          []*Subset                  `json:"globalSubsets"`
	GlobalPolicy           *Policy                    `json:"globalPolicy"`
}

// MeshConfigStatus defines the observed state of MeshConfig
type MeshConfigStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MeshConfig is the Schema for the meshconfigs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=meshconfigs,scope=Namespaced
type MeshConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MeshConfigSpec   `json:"spec,omitempty"`
	Status MeshConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MeshConfigList contains a list of MeshConfig
type MeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MeshConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MeshConfig{}, &MeshConfigList{})
}
