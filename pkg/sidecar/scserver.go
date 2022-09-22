package sidecar

import (
	"context"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/httputil"
	"nmid-v2/pkg/errno"
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

func InflowOutHttpRequestError(w http.ResponseWriter, req *http.Request, code int, err error) error {
	body := make([]byte, 0)
	switch err.(type) {
	case *errno.Errno:
		body = err.(*errno.Errno).Encode()
	default:
		body = model.RequestError.Add(err.Error()).Encode()
		b, _ := httputil.DumpRequest(req, false)
		logger.Errorf("request err: %v, package: \n[ %s ] ", err, string(b))
	}
	w.Header().Set("Content-Type", "application/json;")
	w.WriteHeader(code)
	_, _ = w.Write(body)

	return nil
}

func OutHttpRequestError(w http.ResponseWriter, code int, err error) error {
	body := make([]byte, 0)
	switch err.(type) {
	case *errno.Errno:
		body = err.(*errno.Errno).Encode()
	default:
		body = model.RequestError.Add(err.Error()).Encode()
		logger.Errorf("request err ", err)
	}
	w.Header().Set("Content-Type", "application/json;")
	w.WriteHeader(code)
	_, _ = w.Write(body)

	return nil
}

func OutHttpResponseError(ctx echo.Context, err error) error {
	if err != nil {
		output := make([]byte, 0)
		switch e := err.(type) {
		case *errno.Errno:
			output = e.Encode()
		default:
			output = model.ResponseError.Add(err.Error()).Encode()
		}

		if ctx.Response().Size == 0 {
			ctx.Response().Header().Set("Content-Type", "application/json;")
			ctx.Response().WriteHeader(ctx.Response().Status)
			_, _ = ctx.Response().Write(output)
		}
	}

	return nil
}
