package worker

import "github.com/HughNian/nmid/pkg/metric"

const namespace = "nmid_server"

var (
	WorkerCloseCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "worker",
		Name:      "close_count",
		Help:      "nmid worker close count.",
		Labels:    []string{"worker_ip"},
	})
)
