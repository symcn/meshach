package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// Inject ...
type Inject struct {
	LogLevel string `json:"logLevel,omitempty"`
	Sidecar  string `json:"sidecar"`
}

// Port ...
type Port struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Number   uint32 `json:"number"`
}

// Instance ...
type Instance struct {
	Host   string            `json:"host"`
	Zone   string            `json:"zone"`
	Group  string            `json:"group"`
	Labels map[string]string `json:"labels,omitempty"`
	Port   *Port             `json:"port"`
	Weight uint32            `json:"weight"`
}

// Subset ...
type Subset struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	Policy *Policy           `json:"policy,omitempty"`
}

// Service ...
type Service struct {
	Name      string      `json:"name,omitempty"`
	Ports     []*Port     `json:"ports,omitempty"`
	Instances []*Instance `json:"instances,omitempty"`
	Policy    *Policy     `json:"policy,omitempty"`
	Subsets   []*Subset   `json:"subsets,omitempty"`
}

// Destination ...
type Destination struct {
	Subset string `json:"subset,omitempty"`
	Weight int32  `json:"weight,omitempty"`
}

// Policy ...
type Policy struct {
	LoadBalancer   map[string]string `json:"loadBalancer,omitempty"`
	MaxConnections int32             `json:"maxConnections,omitempty"`
	Timeout        string            `json:"timeout,omitempty"`
	MaxRetries     int32             `json:"maxRetries,omitempty"`
	SourceLabels   []*SourceLabels   `json:"sourceLabels,omitempty"`
}

// SourceLabels ...
type SourceLabels struct {
	Name    string            `json:"name,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Route   []*Destination    `json:"route,omitempty"`
}

// Monitor ...
// type Monitor struct {
// }

// AppMeshConfigSpec ...
type AppMeshConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	AppName  string     `json:"appName,omitempty"`
	Inject   *Inject    `json:"inject"`
	Services []*Service `json:"services"`
	Policy   *Policy    `json:"policy"`
}

// ConfigPhase ...
type ConfigPhase string

// ConfigStatus enumerations
const (
	ConfigStatusUndistributed ConfigPhase = "Undistributed"
	ConfigStatusDistributed   ConfigPhase = "Distributed"
	ConfigStatusDistributing  ConfigPhase = "Distributing"
	ConfigStatusUnknown       ConfigPhase = "Unknown"
)

// SubStatus ...
type SubStatus struct {
	Desired       int  `json:"desired"`
	Distributed   *int `json:"distributed"`
	Undistributed *int `json:"undistributed"`
}

// Status ...
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

// AppMeshConfig is the Schema for the appmeshconfigs API
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
