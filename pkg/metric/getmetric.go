package metric

import (
	"fmt"
	"sort"

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
				}
			}
		}

		if mf.GetName() == "nmid_server_func_func_count" {
			for _, metric := range mf.Metric {
				if metric.Gauge == nil {
					fmt.Println("WARN: metric is not a Gauge type")
					continue
				}

				value := metric.Gauge.GetValue()
				if value > 0 {
					func_num++
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
	ID         string
	Name       string
	Status     string
	CreateTime string
}

type FuncList struct {
	Name       string
	Health     string
	Worker     string
	Host       string
	CreateTime string
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
				var worker WorkerList
				if value > 0 {
					worker.Status = "Online"
				} else {
					worker.Status = "Offline"
				}
				for _, label := range metric.Label {
					if label.GetName() == "worker_name" {
						worker.Name = label.GetValue()
					} else if label.GetName() == "worker_id" {
						worker.ID = label.GetValue()
					} else if label.GetName() == "create_time" {
						worker.CreateTime = label.GetValue()
					}
				}
				workers[worker.Name] = worker

				var funcl FuncList
				if value > 0 {
					funcl.Health = "On"
				} else {
					funcl.Health = "Off"
				}
				for _, label := range metric.Label {
					if label.GetName() == "worker_name" {
						funcl.Worker = label.GetValue()
					} else if label.GetName() == "func_name" {
						funcl.Name = label.GetValue()
					} else if label.GetName() == "worker_ip" {
						funcl.Host = label.GetValue()
					} else if label.GetName() == "create_time" {
						funcl.CreateTime = label.GetValue()
					}
				}

				funcs[funcl.Name] = funcl
			}
		}
	}

	workerList := make([]WorkerList, 0)
	for _, worker := range workers {
		workerList = append(workerList, worker)
	}
	sort.Slice(workerList, func(i, j int) bool {
		return workerList[i].CreateTime < workerList[j].CreateTime
	})

	funcList := make([]FuncList, 0)
	for _, funcl := range funcs {
		funcList = append(funcList, funcl)
	}
	sort.Slice(funcList, func(i, j int) bool {
		return funcList[i].CreateTime < funcList[j].CreateTime
	})

	results["workers"] = workerList
	results["funcs"] = funcList

	return results
}
