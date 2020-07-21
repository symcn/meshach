package option

// AdapterOption ...
type AdapterOption struct {
	EventHandlers EventHandlers
	Registry      Registry
	Configuration Configuration
}

// Registry ...
type Registry struct {
	Type    string
	Address []string
	Timeout int64
}

// Configuration ...
type Configuration struct {
	Type    string
	Address []string
	Timeout int64
}

// EventHandlers ...
type EventHandlers struct {
	// options for kubernetes
	EnableK8s        bool
	IsMultiClusters  bool
	Kubeconfig       string
	ConfigContext    string
	ClusterOwner     string
	ClusterNamespace string
	Namespace        string
	DefaultNamespace string

	// you can add more options for other event handler you will utilize.
	EnableDebugLog bool
}

// DefaultAdapterOption ...
func DefaultAdapterOption() *AdapterOption {
	return &AdapterOption{
		EventHandlers: EventHandlers{
			EnableK8s:        true,
			IsMultiClusters:  false,
			ClusterOwner:     "sym-admin",
			ClusterNamespace: "sym-admin",
			EnableDebugLog:   false,
		},
		Registry: Registry{
			Type:    "zk",
			Address: []string{"127.0.0.1:2181"},
			Timeout: 15,
		},
		Configuration: Configuration{
			Type:    "zk",
			Address: []string{},
			Timeout: 15,
		},
	}
}
