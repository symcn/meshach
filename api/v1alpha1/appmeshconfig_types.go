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

// Service describes Dubbo services, will be registered as ServiceEntries
// in istio's internal service registry.
type Service struct {
	// Must be formatted to conform to the DNS1123 specification.
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	OriginalName string `json:"originalName"`

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
}

// AppMeshConfigSpec ...
type AppMeshConfigSpec struct {
	// Service describes Dubbo services, will be registered as ServiceEntries
	// in istio's internal service registry.
	Services []*Service `json:"services"`
}

// AppMeshConfigStatus defines the observed state of AppMeshConfig
type AppMeshConfigStatus struct {
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
	Phase          ConfigPhase  `json:"phase"`
	Status         *Status      `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// AppMeshConfig is the Schema for the appmeshconfigs API
type AppMeshConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppMeshConfigSpec   `json:"spec,omitempty"`
	Status AppMeshConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppMeshConfigList contains a list of AppMeshConfig
type AppMeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppMeshConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppMeshConfig{}, &AppMeshConfigList{})
}
