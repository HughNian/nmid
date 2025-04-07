package client

import "github.com/HughNian/nmid/pkg/metric"

const namespace = "nmid_server"

var (
	discoveryCount = metric.NewCounterVec(&metric.CounterVecOpts{
		Namespace: namespace,
		Subsystem: "client",
		Name:      "discovery_count",
		Help:      "nmid client discovery count.",
		Labels:    []string{"discovery_name"},
	})
)
