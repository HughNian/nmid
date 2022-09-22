package registry

import (
	"context"
	"strconv"
)

//registry common action

const (
	StatusReceive    = 1
	StatusNotReceive = 2
)

//Config registry config
type Config struct {
	Nodes  []string //registry cluster nodes addr / ectd master nodes addr
	Region string
	Zone   string
	Env    string
	Host   string
}

// metadata common key
const (
	MetaWeight  = "weight"
	MetaCluster = "cluster"
	MetaZone    = "zone"
	MetaColor   = "color"
)

// Instance represents a server the client connects to.
type Instance struct {
	ServiceId   string            `json:"service_id"`
	InFlowAddr  string            `json:"inflow_addr"`
	OutFlowAddr string            `json:"outflow_addr"`
	Region      string            `json:"region"`
	Zone        string            `json:"zone"`
	Env         string            `json:"env"`
	Hostname    string            `json:"hostname"`
	Addrs       []string          `json:"addrs"`
	Version     string            `json:"version"`
	LastTs      int64             `json:"latest_timestamp"`
	Metadata    map[string]string `json:"metadata"`
	Status      int64             `json:"status"`
}

// Resolver resolve naming service
type Resolver interface {
	Fetch() (*InstancesInfo, bool)
	Watch() <-chan struct{}
	Close() error
}

// Registry Register an instance and renew automatically.
type Registry interface {
	Register(ins *Instance) (cancel context.CancelFunc, err error)
	Close() error
}

// Builder resolver builder.
type Builder interface {
	Build(id string) Resolver
	Scheme() string
}

// InstancesInfo instance info.
type InstancesInfo struct {
	Instances map[string][]*Instance `json:"instances"` //zone->[]*Instance
	LastTs    int64                  `json:"latest_timestamp"`
	Scheduler []Zone                 `json:"scheduler"`
}

// Zone zone scheduler info.
type Zone struct {
	Src string           `json:"src"`
	Dst map[string]int64 `json:"dst"`
}

type ReturnWatch struct {
	WType int    `json:"w_type"`
	WKey  string `json:"w_key"`
}

// UseScheduler use scheduler info on instances.
// if instancesInfo contains scheduler info about zone,
// return releated zone's instances weighted by scheduler.
// if not,only zone instances be returned.
func (insInf *InstancesInfo) UseScheduler(zone string) (inss []*Instance) {
	var scheduler struct {
		zone    []string
		weights []int64
	}
	var oriWeights []int64
	for _, sch := range insInf.Scheduler {
		if sch.Src == zone {
			for zone, schWeight := range sch.Dst {
				if zins, ok := insInf.Instances[zone]; ok {
					var totalWeight int64
					for _, ins := range zins {
						var weight int64
						if weight, _ = strconv.ParseInt(ins.Metadata[MetaWeight], 10, 64); weight <= 0 {
							weight = 10
						}
						totalWeight += weight
					}
					oriWeights = append(oriWeights, totalWeight)
					inss = append(inss, zins...)
				}
				scheduler.weights = append(scheduler.weights, schWeight)
				scheduler.zone = append(scheduler.zone, zone)
			}
		}
	}
	if len(inss) == 0 {
		var ok bool
		if inss, ok = insInf.Instances[zone]; ok {
			return
		}
		for _, v := range insInf.Instances {
			inss = append(inss, v...)
		}
		return
	}
	var comMulti int64 = 1
	for _, weight := range oriWeights {
		comMulti *= weight
	}
	var fixWeight = make(map[string]int64, len(scheduler.weights))
	for i, zone := range scheduler.zone {
		fixWeight[zone] = scheduler.weights[i] * comMulti / oriWeights[i]
	}
	for _, ins := range inss {
		var weight int64
		if weight, _ = strconv.ParseInt(ins.Metadata[MetaWeight], 10, 64); weight <= 0 {
			weight = 10
		}
		if fix, ok := fixWeight[ins.Zone]; ok {
			weight = weight * fix
		}
		ins.Metadata[MetaWeight] = strconv.FormatInt(weight, 10)
	}
	return
}
