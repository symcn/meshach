package options

import (
	k8smanager "github.com/mesh-operator/pkg/k8s/manager"
)

// Option ...
type Option struct {
	Address          []string
	Timeout          int64
	ClusterOwner     string
	ClusterNamespace string
	MasterCli        k8smanager.MasterClient
	Registry         Registry
	Configuration    Configuration
}

type Registry struct {
	Type    string
	Address []string
	Timeout int64
}

type Configuration struct {
	Type    string
	Address []string
	Timeout int64
}

// DefaultOption ...
func DefaultOption() *Option {
	return &Option{
		Timeout:          15,
		ClusterOwner:     "sym-admin",
		ClusterNamespace: "sym-admin",
	}
}
