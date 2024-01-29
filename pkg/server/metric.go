package server

import "github.com/HughNian/nmid/pkg/metric"

const namespace = "nmid_server"

var (
	requestCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "client",
		Name:      "request_count",
		Help:      "nmid client requests count.",
		Labels:    []string{"worker_id", "func_name"},
	})

	requestDuring = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "client",
		Name:      "request_during",
		Help:      "nmid client requests during time.",
		Labels:    []string{"worker_id", "func_name"},
	})

	requestDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: namespace,
		Subsystem: "client",
		Name:      "request_duration",
		Help:      "nmid client http requests duration(ms).",
		Labels:    []string{"worker_id", "func_name"},
		Buckets:   []float64{5, 10, 20, 30, 50},
	})
)
