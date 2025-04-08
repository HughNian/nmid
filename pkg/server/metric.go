package server

import "github.com/HughNian/nmid/pkg/metric"

const namespace = "nmid_server"

var (
	requestCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "client",
		Name:      "request_count",
		Help:      "nmid client requests count.",
		Labels:    []string{"worker_name", "func_name"},
	})

	requestDuring = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "client",
		Name:      "request_during",
		Help:      "nmid client requests during time.",
		Labels:    []string{"worker_name", "func_name"},
	})

	requestDuration = metric.NewHistogramVec(&metric.HistogramVecOpts{
		Namespace: namespace,
		Subsystem: "client",
		Name:      "request_duration",
		Help:      "nmid client http requests duration(ms).",
		Labels:    []string{"worker_name", "func_name"},
		Buckets:   []float64{5, 10, 20, 30, 50},
	})

	WorkerCloseCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "worker",
		Name:      "close_count",
		Help:      "nmid worker close count.",
		Labels:    []string{"worker_ip"},
	})

	WorkerFuncCount = metric.NewGaugeVec(&metric.GaugeVecOpts{
		Namespace: namespace,
		Subsystem: "worker",
		Name:      "func_count",
		Help:      "nmid worker func count.",
		Labels:    []string{"worker_id", "worker_name", "worker_ip", "func_name"},
	})

	WorkerFuncSuccessCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "worker",
		Name:      "func_success",
		Help:      "nmid worker func do job success count.",
		Labels:    []string{"worker_name", "func_name"},
	})

	WorkerFuncFailCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "worker",
		Name:      "func_fail",
		Help:      "nmid worker func do job fail count.",
		Labels:    []string{"worker_name", "func_name"},
	})

	FuncCount = metric.NewGaugeVec(&metric.GaugeVecOpts{
		Namespace: namespace,
		Subsystem: "func",
		Name:      "func_count",
		Help:      "nmid all func count.",
		Labels:    []string{"func_name"},
	})

	FuncSuccessCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "func",
		Name:      "func_success",
		Help:      "nmid func do job success count.",
		Labels:    []string{"func_name"},
	})

	FuncFailCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "func",
		Name:      "func_fail",
		Help:      "nmid func do job fail count.",
		Labels:    []string{"func_name"},
	})

	ThreatIpCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "ip",
		Name:      "threat_ip",
		Help:      "nmid threat ip count.",
		Labels:    []string{"ip", "zone", "country", "prov", "city", "lat", "lon"},
	})
)
