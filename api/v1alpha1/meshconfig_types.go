/*


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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StringMatchType ...
type StringMatchType string

// StringMatchType enum
var (
	Exact  StringMatchType = "exact"
	Prefix StringMatchType = "prefix"
	Regex  StringMatchType = "regex"
)

// Subset is a subset of endpoints of a service. Subset can be used for
// scenarios like A/B testing, or routing to a specific version of a service.
type Subset struct {
	// Must be formatted to conform to the DNS1123 specification.
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// One or more labels are typically required to identify the subset destination.
	// e.g. {"group": "blue"}
	Labels map[string]string `json:"labels"`

	// Traffic policies defined at the service-level can be overridden at a subset-level.
	// NOTE: Policies specified for subsets will not take effect until a route rule explicitly
	// sends traffic to this subset.
	Policy *Policy `json:"policy,omitempty"`

	// Whether the subset is defined as a canary group
	IsCanary bool `json:"isCanary,omitempty"`
}

// MeshConfigSpec defines the desired state of MeshConfig
type MeshConfigSpec struct {
	// +kubebuilder:validation:Optional
	MatchSourceLabelKeys []string `json:"matchSourceLabelKeys"`

	// +kubebuilder:validation:Optional
	WorkloadEntryLabelKeys []string `json:"workloadEntryLabelKeys"`

	// +kubebuilder:validation:Optional
	MeshLabelsRemap map[string]string `json:"meshLabelsRemap"`

	// +kubebuilder:validation:Optional
	ExtractedLabels []string `json:"extractedLabels"`

	// +kubebuilder:validation:Optional
	SidecarSelectLabel string `json:"sidecarSelectLabel"`

	// +kubebuilder:validation:Optional
	SidecarDefaultHosts []string `json:"sidecarDefaultHosts"`

	GlobalSubsets []*Subset `json:"globalSubsets"`
	GlobalPolicy  *Policy   `json:"globalPolicy"`
}

// MeshConfigStatus defines the observed state of MeshConfig
type MeshConfigStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// MeshConfig is the Schema for the meshconfigs API
type MeshConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MeshConfigSpec   `json:"spec,omitempty"`
	Status MeshConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MeshConfigList contains a list of MeshConfig
type MeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MeshConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MeshConfig{}, &MeshConfigList{})
}

// In ...
func (s *Subset) In(list []*Subset) bool {
	for _, item := range list {
		if item.Name == s.Name {
			return true
		}
	}
	return false
}
