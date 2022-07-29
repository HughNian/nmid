package sidecar

import (
	"context"
	"nmid-v2/pkg/logger"
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

func NewScServer(doneCtx context.Context, config model.ServerConfig) *ScServer {
	sc := &ScServer{
		doneCtx: doneCtx,
	}
	sc.inflow = NewInflowServer(sc, config)
	sc.outflow = NewOutflowServer(sc, config)

	logger.Info("sidecar inflow address: ", config.SideCar.InflowAddr.BindAddress)
	logger.Info("sidecar outflow address: ", config.SideCar.OutflowAddr.BindAddress)

	return sc
}

func (sc *ScServer) StartScServer() {
	logger.Info("sidecar car start ok")

	go sc.inflow.StartInflow()
	go sc.outflow.StartOutflow()
}
