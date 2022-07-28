package sidecar

import (
	"context"
	"nmid-v2/pkg/model"
	"nmid-v2/pkg/registry"
	"sync"
)

//sidecar server, just http server now
//use two http ports, inflow port & outflow port

const (
	RequestBodyLimit = 1024 * 1024 * 4
	ResponseBodySize = 1024 * 1024 * 10
)

type ScServer struct {
	sync.RWMutex

	registry registry.Registry
	inflow   *inflowServer
	outflow  *outflowServer

	doneCtx context.Context
}

func NewScServer(config model.ServerConfig) *ScServer {
	sc := &ScServer{
		inflow:  NewInflowServer(config),
		outflow: NewOutflowServer(config),
	}

	return sc
}

func (sc *ScServer) StartScServer() {
	go sc.inflow.StartInflow()
	go sc.outflow.StartOutflow()
}
