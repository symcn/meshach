package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Inject set the relevant parameters of the data plane.
type Inject struct {
	// The log level of sidecar
	// +kubebuilder:validation:Enum=error;warning;info;debug
	LogLevel string `json:"logLevel,omitempty"`

	// Set the sidecar type, must one of mosn|envoy
	// +kubebuilder:validation:Enum=mosn;envoy
	Sidecar string `json:"sidecar"`
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

// Instance describes the properties of a specific instance of a service.
type Instance struct {
	// Host associated with the network endpoint without the port.
	// +kubebuilder:validation:MaxLength=16
	// +kubebuilder:validation:MinLength=8
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
}

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

	// One or more labels that constrain the applicability of a rule to workloads with
	// the given labels.
	SourceLabels []*SourceLabels `json:"sourceLabels,omitempty"`
}

// SourceLabels is one or more labels that constrain the applicability of a rule to workloads with
// the given labels.
type SourceLabels struct {
	// The name of specific subset.
	Name string `json:"name,omitempty"`

	// One or more labels that constrain the applicability of a rule to workloads with
	// the given labels.
	Labels map[string]string `json:"labels,omitempty"`

	// The HTTP route match headers.
	Headers map[string]string `json:"headers,omitempty"`

	// Each routing rule is associated with one or more service versions.
	Route []*Destination `json:"route,omitempty"`
}

// AppMeshConfigSpec ...
type AppMeshConfigSpec struct {
	// Service describes Dubbo services, will be registered as ServiceEntries
	// in istio's internal service registry.
	Services []*Service `json:"services"`
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
	ServiceEntry    *SubStatus `json:"serviceEntry"`
	WorkloadEntry   *SubStatus `json:"workloadEntry"`
	VirtualService  *SubStatus `json:"virtualService"`
	DestinationRule *SubStatus `json:"destinationRule"`
}

// AppMeshConfigStatus defines the observed state of AppMeshConfig
type AppMeshConfigStatus struct {
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
	Phase          ConfigPhase  `json:"phase"`
	Status         *Status      `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppMeshConfig is the Schema for the appmeshconfigs API.
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=appmeshconfigs,scope=Namespaced
type AppMeshConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppMeshConfigSpec   `json:"spec,omitempty"`
	Status AppMeshConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AppMeshConfigList contains a list of AppMeshConfig
type AppMeshConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppMeshConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppMeshConfig{}, &AppMeshConfigList{})
}
