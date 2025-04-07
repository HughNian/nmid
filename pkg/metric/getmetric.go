package metric

import (
	prom "github.com/prometheus/client_golang/prometheus"
)

func GetDiscoveryWorkerFuncNum() map[string]interface{} {
	mfs, _ := prom.DefaultGatherer.Gather()

	var (
		discovery_num = 0
		worker_num    = 0
		func_num      = 0
	)
	results := make(map[string]interface{})
	worker_funcs := make(map[string]int)

	for _, mf := range mfs {
		if mf.GetName() == "nmid_server_client_discovery_count" {
			for _, metric := range mf.Metric {
				for _, label := range metric.Label {
					if label.GetName() == "discovery_name" {
						discovery_num++
					}

				}
			}
		}

		if mf.GetName() == "nmid_server_worker_func_count" {
			for _, metric := range mf.Metric {
				// if metric.Gauge == nil {
				// 	fmt.Println("WARN: metric is not a Gauge type")
				// 	continue
				// }
				// count := metric.Gauge.GetValue()

				// var workerName, funcName string
				for _, label := range metric.Label {
					if label.GetName() == "worker_name" {
						workerName := label.GetValue()
						if worker_funcs[workerName] == 0 {
							worker_num++
						}

						worker_funcs[label.GetValue()]++
					}
					if label.GetName() == "func_name" {
						// funcName = label.GetValue()
						func_num++
					}
				}

				// fmt.Printf("Gauge [%s/%s] count: %.2f\n",
				// 	funcName,
				// 	workerName,
				// 	count)
			}
		}
	}

	results["discovery_num"] = discovery_num
	results["worker_num"] = worker_num
	results["func_num"] = func_num
	return results
}
