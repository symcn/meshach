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

// SourceLabels is one or more labels that constrain the applicability of a rule to workloads with
// the given labels.
// type SourceLabels struct {
// 	// The name of specific subset.
// 	Name string `json:"name,omitempty"`

// 	// One or more labels that constrain the applicability of a rule to workloads with
// 	// the given labels.
// 	Labels map[string]string `json:"labels,omitempty"`

// 	// The HTTP route match headers.
// 	Headers map[string]string `json:"headers,omitempty"`
// }

// Instance describes the properties of a specific instance of a service.
type Instance struct {
	// Host associated with the network endpoint without the port.
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=7
	Host string `json:"host"`

	// The parameters of Dubbo service
	Labels map[string]string `json:"labels"`

	// Port describes the properties of a specific port of a service.
	// The Dubbo service port registered with MOSN is 20882,
	// otherwize the native Dubbo service port is 20880.
	Port *Port `json:"port"`

	// The traffic weight of this instance.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Weight uint32 `json:"weight,omitempty"`
}

// Port describes the properties of a specific port of a service.
type Port struct {
	// Label assigned to the port.
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// The protocol exposed on the port.
	// MUST BE HTTP TO ROUTE DUBBO SERVICE.
	// +kubebuilder:validation:Enum=HTTP;HTTPS;TCP
	Protocol string `json:"protocol"`

	// A valid non-negative integer port number.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Number uint32 `json:"number"`
}

// ConfiguredServiceSpec defines the desired state of ConfiguredService
type ConfiguredServiceSpec struct {
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:MinLength=1
	OriginalName string `json:"originalName"`

	// A list describes all registered instances of this service.
	Instances []*Instance `json:"instances,omitempty"`

	// A list describes the properties of all ports of this service.
	// The Dubbo service port registered with MOSN is 20882,
	// otherwize the native Dubbo service port is 20880.
	// +kubebuilder:validation:MinItems=1
	// Ports []*Port `json:"ports,omitempty"`

	// Subsets defined all sebsets of the current service.
	// Subsets []*Subset `json:"subsets,omitempty"`

	// The Generation of MeshConfig, which to reconcile AppMeshConfig when MeshConfig changes.
	MeshConfigGeneration int64 `json:"meshConfigGeneration,omitempty"`
}

// ConfiguredServiceStatus defines the observed state of ConfiguredService
type ConfiguredServiceStatus struct {
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// ConfiguredService is the Schema for the configuredservices API
type ConfiguredService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfiguredServiceSpec   `json:"spec,omitempty"`
	Status ConfiguredServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ConfiguredServiceList contains a list of ConfiguredService
type ConfiguredServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ConfiguredService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ConfiguredService{}, &ConfiguredServiceList{})
}
