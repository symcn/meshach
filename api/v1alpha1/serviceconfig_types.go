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

// ServiceConfigSpec defines the desired state of ServiceConfig
type ServiceConfigSpec struct {
	// The settings used to reroute the sourceLabels traffic when all of the previously
	// subsets instances are available.
	RerouteOption *RerouteOption `json:"rerouteOption,omitempty"`

	// The settings used to reroute canary deployment traffic when all of the previously
	// subsets instances are available.
	CanaryRerouteOption *RerouteOption `json:"canaryRerouteOption,omitempty"`

	// The Generation of MeshConfig, which to reconcile AppMeshConfig when MeshConfig changes.
	MeshConfigGeneration int64 `json:"meshConfigGeneration,omitempty"`

	// Traffic policies of service-level
	Policy *Policy `json:"policy,omitempty"`

	// Each routing rule is associated with one or more service versions.
	Route []*Destination `json:"route,omitempty"`

	// A list describes all registered instances config of this service.
	Instances []*InstanceConfig
}

// Destination indicates the network addressable service to which the request/connection
// will be sent after processing a routing rule.
type Destination struct {
	// The name of a subset within the service. Applicable only to services
	// within the mesh. The subset must be defined in a corresponding DestinationRule.
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=1
	Subset string `json:"subset"`

	// The proportion of traffic to be forwarded to the service
	// version. (0-100). Sum of weights across destinations SHOULD BE == 100.
	// If there is only one destination in a rule, the weight value is assumed to be 100.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Weight int32 `json:"weight,omitempty"`
}

// Policy defines load balancing, retry, and other policies for a service.
type Policy struct {
	// Load balancing is a way of distributing traffic between multiple hosts within
	// a single upstream cluster in order to effectively make use of available resources.
	// There are many different ways of accomplishing this, like ROUND_ROBIN, LEAST_CONN
	// RANDOM and RASSTHROUGN
	LoadBalancer map[string]string `json:"loadBalancer,omitempty"`

	// Maximum number of HTTP1 connections to a destination host. Default 2^32-1.
	MaxConnections int32 `json:"maxConnections,omitempty"`

	// Connection timeout. format: 1h/1m/1s/1ms. MUST BE >=1ms. Default is 10s.
	Timeout string `json:"timeout,omitempty"`

	// Maximum number of retries that can be outstanding to all hosts in a cluster at a given time.
	// Defaults to 2^32-1.
	MaxRetries int32 `json:"maxRetries,omitempty"`
}

// RerouteOption define the re-routing policy when all of the previously sebsets instances
// are available.
type RerouteOption struct {
	// The rerouting strategy currently includes only Unchangeable, Polling, Random, and Specific.
	//
	// 1. Unchangeable: Does not reroute when all instances of the originally
	// specified subset are offline.
	//
	// 2. RoundRobin: Access the remaining available subsets when all instances
	// of a sourcelabels originally specified are offline using Round-Robin.
	//
	// 3. Random: When all instances of the originally specified subset for a
	// sourcelabels are offline, random access to the remaining available subsets.
	//
	// 4. Specific: Routes using the specified mapping when all instances of the
	// originally specified subset are offline.
	ReroutePolicy ReroutePolicy `json:"reroutePolicy"`

	// This map only takes effect when 'ReroutePolicy' is specified to 'Specific',
	// each sourceLabels can specify multiple accessible subsets and weight.
	SpecificRoute map[string]Destinations `json:"specificRoute,omitempty"`
}

// InstanceConfig describes the configs of a specific instance of a service.
type InstanceConfig struct {
	// Host associated with the network endpoint without the port.
	// +kubebuilder:validation:MaxLength=16
	// +kubebuilder:validation:MinLength=8
	Host string `json:"host"`

	// Port describes the properties of a specific port of a service.
	// The Dubbo service port registered with MOSN is 20882,
	// otherwize the native Dubbo service port is 20880.
	Port *Port `json:"port"`

	// The traffic weight of this instance.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Weight uint32 `json:"weight,omitempty"`
}

// Destinations ...
type Destinations []*Destination

// ReroutePolicy ...
type ReroutePolicy string

// The enumerations of ReroutePolicy
const (
	Default      ReroutePolicy = "Default"
	Unchangeable ReroutePolicy = "Unchangeable"
	Specific     ReroutePolicy = "Specific"
)

// SubStatus describes the destribution status of the individual istio's CR.
type SubStatus struct {
	// Total number of desired configuration files.
	Desired int `json:"desired"`

	// Total number of configuration files distributed.
	Distributed *int `json:"distributed"`

	// Total number of configuration files undistributed.
	Undistributed *int `json:"undistributed"`
}

// Status is a collection of all SubStatus.
type Status struct {
	ServiceEntry    *SubStatus `json:"serviceEntry,omitempty"`
	WorkloadEntry   *SubStatus `json:"workloadEntry,omitempty"`
	VirtualService  *SubStatus `json:"virtualService,omitempty"`
	DestinationRule *SubStatus `json:"destinationRule,omitempty"`
}

// ConfigPhase describes the phase of the configuration file destribution.
type ConfigPhase string

// ConfigStatus enumerations
const (
	ConfigStatusUndistributed ConfigPhase = "Undistributed"
	ConfigStatusDistributed   ConfigPhase = "Distributed"
	ConfigStatusDistributing  ConfigPhase = "Distributing"
	ConfigStatusUnknown       ConfigPhase = "Unknown"
)

// ServiceConfigStatus defines the observed state of ServiceConfig
type ServiceConfigStatus struct {
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
	Phase          ConfigPhase  `json:"phase,omitempty"`
	Status         *Status      `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceConfig is the Schema for the serviceconfigs API
type ServiceConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceConfigSpec   `json:"spec,omitempty"`
	Status ServiceConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceConfigList contains a list of ServiceConfig
type ServiceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceConfig{}, &ServiceConfigList{})
}
