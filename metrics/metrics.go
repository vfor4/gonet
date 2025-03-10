package metrics

import (
	"flag"

	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

var (
	Namespace = flag.String("namespace", "web", "metrics namespace")
	Subsystem = flag.String("subsystem", "server1", "metrics subsystem")
	// counters
	Requests metrics.Counter = prometheus.NewCounterFrom(prom.CounterOpts{
		Namespace: *Namespace,
		Subsystem: *Subsystem,
		Name:      "request_counter",
		Help:      "Count the number of requests",
	}, []string{})
	WriteErrors metrics.Counter = prometheus.NewCounterFrom(prom.CounterOpts{
		Namespace: *Namespace,
		Subsystem: *Subsystem,
		Name:      "errors_counter",
		Help:      "Count the number of written errors",
	}, []string{})

	//gauges
	OpenConnections metrics.Gauge = prometheus.NewGaugeFrom(
		prom.GaugeOpts{
			Namespace: *Namespace,
			Subsystem: *Subsystem,
			Name:      "open_connections",
			Help:      "Total open connections",
		}, []string{},
	)

	RequestDurationHistogram metrics.Histogram = prometheus.NewHistogramFrom(
		prom.HistogramOpts{
			Namespace: *Namespace,
			Subsystem: *Subsystem,
			Buckets: []float64{
				0.0000001, 0.0000002, 0.0000003, 0.0000004, 0.0000005,
				0.000001, 0.0000025, 0.000005, 0.0000075, 0.00001,
				0.0001, 0.001, 0.01,
			},
			Name: "request_duration_histogram_seconds",
			Help: "Total duration of all requests",
		},
		[]string{},
	)

	RequesetDurationSumary metrics.Histogram = prometheus.NewSummaryFrom(
		prom.SummaryOpts{
			Namespace: *Namespace,
			Subsystem: *Subsystem,
			Name:      "request_duration_summary_seconds",
			Help:      "Total duration of all requests",
		}, []string{},
	)
)
