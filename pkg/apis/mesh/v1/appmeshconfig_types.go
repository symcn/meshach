package v1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
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
	Port     int32  `json:"port"`
}

// Instance ...
type Instance struct {
	Host   string            `json:"host"`
	Zone   string            `json:"zone"`
	Group  string            `json:"group"`
	Labels map[string]string `json:"labels,omitempty"`
	Port   *Port             `json:"port"`
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
	AppName   string      `json:"appName,omitempty"`
	Ports     []*Port     `json:"ports,omitempty"`
	Instances []*Instance `json:"instances,omitempty"`
	Policy    *Policy     `json:"policy,omitempty"`
	Subsets   []*Subset   `json:"subsets,omitempty"`
}

// Destination ...
type Destination struct {
	Subset string `json:"subset,omitempty"`
	Weight string `json:"weight,omitempty"`
}

// Policy ...
type Policy struct {
	LoadBalancer   map[string]string `json:"loadBalancer,omitempty"`
	MaxConnections int32             `json:"maxConnections,omitempty"`
	Timeout        string            `json:"timeout,omitempty"`
	MaxRetries     int32             `json:"maxRetries,omitempty"`
	Route          []*Destination    `json:"route,omitempty"`
}

// Monitor ...
// type Monitor struct {
// }

// AppMeshConfigSpec ...
type AppMeshConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	Inject   *Inject    `json:"inject"`
	Services []*Service `json:"services"`
	Policy   *Policy    `json:"policy"`
}

// ConfigStatus ...
type ConfigStatus string

// ConfigStatus enumerations
const (
	ConfigStatusUndistributed ConfigStatus = "Undistributed"
	ConfigStatusDistributed   ConfigStatus = "Distributed"
	ConfigStatusUpdating      ConfigStatus = "Updating"
	ConfigStatusUnknown       ConfigStatus = "Unknown"
)

// AppMeshConfigStatus defines the observed state of AppMeshConfig
type AppMeshConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// +optional
	ObservedGeneration int64                   `json:"observedGeneration,omitempty"`
	LastUpdateTime     *metav1.Time            `json:"lastUpdateTime,omitempty"`
	Status             ConfigStatus            `json:"status"`
	XdsStatus          map[string]ConfigStatus `json:"xdsStatus"`
	Conditions         status.Conditions       `json:"conditions"`
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
