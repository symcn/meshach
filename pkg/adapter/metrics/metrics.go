package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// promauto
var (
	SynchronizedServiceCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mesh_adapter_syn_service_total",
		Help: "The total number of services which has been synchronized",
	})

	DeletedServiceCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mesh_adapter_delete_service_total",
		Help: "The total number of services which has been Deleted",
	})

	SynchronizedInstanceCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mesh_adapter_syn_instance_total",
		Help: "The total number of instances which has been synchronized",
	})

	ReplacingInstancesHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "mesh_adapter_replace_instances_seconds_bucket",
		Help:    "The latency of replacing instances",
		Buckets: []float64{0.01, 0.1, 0.5, 1, 5, 10},
	})

	ReplacedAccessorInstancesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mesh_adapter_replace_accessor_instances_total",
		Help: "The total number of accessor instances which has been replaced",
	})

	AddedConfigurationCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mesh_adapter_add_configuration_total",
		Help: "The total number of configuration which has been added",
	})

	ChangedConfigurationCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mesh_adapter_change_configuration_total",
		Help: "The total number of configuration which has been changed",
	})

	DeletedConfigurationCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mesh_adapter_delete_configuration_total",
		Help: "The total number of configuration which has been deleted",
	})

	ChangingConfigurationHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "mesh_adapter_change_configuration_seconds_bucket",
		Help:    "The latency of changing configuration",
		Buckets: []float64{0.01, 0.1, 0.5, 1, 5, 10},
	})

	PathCacheLengthGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mesh_adapter_path_cache_gauge",
			Help: "The size of path cache",
		},
		[]string{"path"},
	)
)
