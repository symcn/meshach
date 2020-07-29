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
