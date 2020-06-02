package metrics

import (
	"fmt"

	ocprom "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	// ServerLatencyView ...
	ServerLatencyView = &view.View{
		Name:        "opencensus.io/http/server/latency",
		Description: "Latency distribution of HTTP requests",
		TagKeys:     []tag.Key{ochttp.Path},
		Measure:     ochttp.ServerLatency,
		Aggregation: ochttp.DefaultLatencyDistribution,
	}
	// ServerResponseCountByStatusCode ...
	ServerResponseCountByStatusCode = &view.View{
		Name:        "opencensus.io/http/server/response_count_by_status_code",
		Description: "Server response count by status code",
		TagKeys:     []tag.Key{ochttp.Path, ochttp.StatusCode},
		Measure:     ochttp.ServerLatency,
		Aggregation: view.Count(),
	}
)

// OcPrometheus ...
type OcPrometheus struct {
	Exporter *ocprom.Exporter
}

// NewOcPrometheus ...
func NewOcPrometheus() (*OcPrometheus, error) {
	registry := prometheus.DefaultRegisterer.(*prometheus.Registry)

	exporter, err := ocprom.NewExporter(ocprom.Options{Registry: registry})
	if err != nil {
		err = fmt.Errorf("could not set up prometheus exporter: %v", err)
		return nil, err
	}

	p := &OcPrometheus{
		Exporter: exporter,
	}
	view.RegisterExporter(exporter)

	return p, nil
}

// RegisterGinView ...
func RegisterGinView() error {
	// Register stat views
	err := view.Register(
		// Gin (HTTP) stats
		ochttp.ServerRequestCountView,
		ochttp.ServerRequestBytesView,
		ochttp.ServerResponseBytesView,
		ServerLatencyView,
		ochttp.ServerRequestCountByMethod,
		ServerResponseCountByStatusCode,
	)
	if err != nil {
		panic(err)
	}

	return nil
}
