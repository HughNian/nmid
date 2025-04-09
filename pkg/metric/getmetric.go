package metric

import (
	"fmt"

	prom "github.com/prometheus/client_golang/prometheus"
)

// 注册中心，worker，function数量
func GetDiscoveryWorkerFuncNum() map[string]int {
	mfs, _ := prom.DefaultGatherer.Gather()
	results := make(map[string]int)
	worker_funcs := make(map[string]int)

	var (
		discovery_num = 0
		worker_num    = 0
		func_num      = 0
	)
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
				if metric.Gauge == nil {
					fmt.Println("WARN: metric is not a Gauge type")
					continue
				}

				value := metric.Gauge.GetValue()
				for _, label := range metric.Label {
					if label.GetName() == "worker_name" {
						workerName := label.GetValue()
						if value > 0 && worker_funcs[workerName] == 0 {
							worker_num++
						}

						if value > 0 {
							worker_funcs[label.GetValue()]++
						}
					}
					if label.GetName() == "func_name" {
						if value > 0 {
							func_num++
						}
					}
				}
			}
		}
	}

	results["discovery_num"] = discovery_num
	results["worker_num"] = worker_num
	results["func_num"] = func_num
	return results
}

// worker调用成功，调用失败，关闭数量
func GetSuccesFailCloseNum() map[string]float64 {
	mfs, _ := prom.DefaultGatherer.Gather()
	results := make(map[string]float64)

	var success_num, fail_num, close_num float64
	for _, mf := range mfs {
		if mf.GetName() == "nmid_server_worker_func_success" {
			for _, metric := range mf.Metric {
				if metric.Counter == nil {
					fmt.Println("WARN: metric is not a Counter type")
					continue
				}

				success_num += metric.Counter.GetValue()
			}
		}

		if mf.GetName() == "nmid_server_worker_func_fail" {
			for _, metric := range mf.Metric {
				if metric.Counter == nil {
					fmt.Println("WARN: metric is not a Counter type")
					continue
				}

				fail_num += metric.Counter.GetValue()
			}
		}

		if mf.GetName() == "nmid_server_worker_close_count" {
			for _, metric := range mf.Metric {
				if metric.Counter == nil {
					fmt.Println("WARN: metric is not a Counter type")
					continue
				}

				close_num += metric.Counter.GetValue()
			}
		}
	}

	results["success_num"] = success_num
	results["fail_num"] = fail_num
	results["close_num"] = close_num
	return results
}

type WorkerList struct {
	ID     string
	Name   string
	Status string
}

type FuncList struct {
	Name   string
	Health string
	Worker string
	Host   string
}

func GetWorkersFuncs() map[string]interface{} {
	mfs, _ := prom.DefaultGatherer.Gather()
	workers := make(map[string]WorkerList)
	funcs := make(map[string]FuncList)
	results := make(map[string]interface{})

	for _, mf := range mfs {
		if mf.GetName() == "nmid_server_worker_func_count" {
			for _, metric := range mf.Metric {
				if metric.Gauge == nil {
					fmt.Println("WARN: metric is not a Gauge type")
					continue
				}

				value := metric.Gauge.GetValue()
				if value != 0 {
					var worker WorkerList
					worker.Status = "Online"

					var funcl FuncList
					funcl.Health = "OK"

					for _, label := range metric.Label {
						if label.GetName() == "worker_name" {
							worker.Name = label.GetValue()
							funcl.Worker = label.GetValue()
						} else if label.GetName() == "worker_id" {
							worker.ID = label.GetValue()
						} else if label.GetName() == "func_name" {
							funcl.Name = label.GetValue()
						} else if label.GetName() == "worker_ip" {
							funcl.Host = label.GetValue()
						}
					}
					workers[worker.Name] = worker
					funcs[funcl.Name] = funcl
				}
			}
		}
	}

	workerList := make([]WorkerList, 0)
	for _, worker := range workers {
		workerList = append(workerList, worker)
	}

	funcList := make([]FuncList, 0)
	for _, funcl := range funcs {
		funcList = append(funcList, funcl)
	}

	results["workers"] = workerList
	results["funcs"] = funcList

	return results
}
