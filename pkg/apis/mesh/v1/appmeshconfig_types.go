package v1

import (
	networking "istio.io/api/networking/v1alpha3"
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
	Host       string            `json:"host"`
	Zone       string            `json:"zone"`
	Group      string            `json:"group"`
	Labels     map[string]string `json:"labels,omitempty"`
	Dynamic    bool              `json:"dynamic,omitempty"`
	Weight     string            `json:"weight,omitempty"`
	Methods    []string          `json:"methods,omitempty"`
	ProCode    string            `json:"pro_code,omitempty"`
	Release    string            `json:"release,omitempty"`
	SdkVersion string            `json:"sdk_version,omitempty"`
	Side       string            `json:"side,omitempty"`
	Timeout    int32             `json:"timeout,omitempty"`
	Retry      int32             `json:"retry,omitempty"`
	Timestamp  int64             `json:"timestamp,omitempty"`
}

// Service ...
type Service struct {
	Name        string      `json:"name"`
	Application string      `json:"application,omitempty"`
	Ports       []*Port     `json:"ports"`
	Instances   []*Instance `json:"instances"`
}

// Policy ...
type Policy struct {
	LoadBalancer      *networking.LoadBalancerSettings `json:"load_balancer,omitempty"`
	MaxConnections    int32                            `json:"max_connections,omitempty"`
	ConnectionTimeout string                           `json:"connection_timeout,omitempty"`
	MaxRetries        int32                            `json:"max_retries,omitempty"`
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

type ConfigStatus string

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
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`
	Status         ConfigStatus
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
