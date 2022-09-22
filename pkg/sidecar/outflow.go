package sidecar

import (
	"context"
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"nmid-v2/pkg/logger"
	"nmid-v2/pkg/model"
	"time"
)

type outflowServer struct {
	*ScServer
	httpServer *echo.Echo
	httpProxy  *httpProxy
}

func NewOutflowServer(sc *ScServer, config model.ServerConfig) *outflowServer {
	opt := config.SideCar.OutflowAddr
	opt.RequestBodySize = RequestBodyLimit
	opt.ResponseBodySize = ResponseBodySize
	opt.ReadTimeout = 60 * time.Second
	opt.WriteTimeout = 60 * time.Second
	opt.IdleTimeout = 60 * time.Second

	proxy, err := NewHttpProxy(opt)
	if nil != err {
		logger.Error("new http proxy err ", err.Error())
		return nil
	}

	return &outflowServer{
		ScServer:  sc,
		httpProxy: proxy,
	}
}

func (os *outflowServer) StartOutflow() {
	os.httpServer = echo.New()
	os.httpServer.HideBanner = true
	os.httpServer.HidePort = true

	//offline
	ctx, cancel := context.WithCancel(os.doneCtx)
	if err := recover(); nil != err {
		select {
		case <-ctx.Done():
			ctxs, _ := context.WithTimeout(context.Background(), 5*time.Second)
			err := os.httpProxy.Server.Shutdown(ctxs)
			if err != nil {
				logger.Warnf("inflow server offline %v", err)
			}
		}
	}
	os.httpProxy.Server.RegisterOnShutdown(func() {
		cancel()
	})

	//do proxy
	middls := os.doMiddle()
	os.httpServer.Any("*", os.doOutflowProxy, middls...)

	err := os.httpServer.StartServer(os.httpProxy.Server)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatal(err)
	}
}

func (os *outflowServer) doMiddle(middls ...echo.MiddlewareFunc) []echo.MiddlewareFunc {
	if len(middls) == 0 {
		return []echo.MiddlewareFunc{}
	}

	return middls
}

func (os *outflowServer) doOutflowProxy(c echo.Context) error {

	return nil
}
