package utils

// ControllerOption ...
type ControllerOption struct {
	HTTPAddress             string
	MetricsEnabled          bool
	SyncPeriod              int32
	LeaderElectionNamespace string
	LeaderElectionID        string
	EnableLeaderElection    bool
	GinLogEnabled           bool
	GinLogSkipPath          []string
	PprofEnabled            bool
	GoroutineThreshold      int

	// The exact zone code of current cluster, gz01/rz01/rz02
	Zone string

	// Dubbo proxy settings
	ProxyHost          string
	ProxyAttempts      int32
	ProxyPerTryTimeout int64
	ProxyRetryOn       string
}

// DefaultControllerOption ...
func DefaultControllerOption() *ControllerOption {
	return &ControllerOption{
		HTTPAddress:             ":8080",
		SyncPeriod:              120,
		MetricsEnabled:          true,
		GinLogEnabled:           true,
		GinLogSkipPath:          []string{"/ready", "/live"},
		EnableLeaderElection:    true,
		LeaderElectionID:        "mesh-operator-lock",
		LeaderElectionNamespace: "sym-admin",
		PprofEnabled:            true,
		GoroutineThreshold:      1000,
		ProxyHost:               "mosn.io.dubbo.proxy",
		ProxyAttempts:           3,
		ProxyPerTryTimeout:      2,
		ProxyRetryOn:            "gateway-error,connect-failure,refused-stream",
		Zone:                    "gz01",
	}
}
