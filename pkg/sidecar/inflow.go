package sidecar

import (
	"context"
	"errors"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"nmid-v2/pkg/logger"
	"nmid-v2/pkg/model"
	"time"
)

type inflowServer struct {
	*ScServer
	httpServer *echo.Echo
	httpProxy  *httpProxy
}

func NewInflowServer(sc *ScServer, config model.ServerConfig) *inflowServer {
	opt := config.SideCar.InflowAddr
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
	return &inflowServer{
		ScServer:  sc,
		httpProxy: proxy,
	}
}

func (is *inflowServer) StartInflow() {
	is.httpServer = echo.New()
	is.httpServer.HideBanner = true
	is.httpServer.HidePort = true

	//offline
	ctx, cancel := context.WithCancel(is.doneCtx)
	if err := recover(); nil != err {
		select {
		case <-ctx.Done():
			ctxs, _ := context.WithTimeout(context.Background(), 5*time.Second)
			err := is.httpProxy.Server.Shutdown(ctxs)
			if err != nil {
				logger.Warnf("inflow server offline %v", err)
			}
		}
	}
	is.httpProxy.Server.RegisterOnShutdown(func() {
		cancel()
	})

	//do proxy
	middls := is.doMiddle()
	is.httpServer.Any("*", is.doInflowProxy, middls...)

	err := is.httpServer.StartServer(is.httpProxy.Server)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatal(err)
	}
}

func (is *inflowServer) doMiddle(middls ...echo.MiddlewareFunc) []echo.MiddlewareFunc {
	if len(middls) == 0 {
		return []echo.MiddlewareFunc{}
	}

	return middls
}

func (is *inflowServer) doInflowProxy(c echo.Context) error {
	req := c.Request()

	// set proxy scheme host
	req.URL.Scheme = is.httpProxy.Options.TargetProtocolType
	req.Host, req.URL.Host = is.httpProxy.Options.TargetAddress, is.httpProxy.Options.TargetAddress
	// do proxy
	resp, err := is.httpProxy.Proxy.Transport.RoundTrip(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
			return nil
		}

		return OutHttpResponseError(c, model.RequestError.Add(err.Error()))
	}

	defer resp.Body.Close()

	// request body size limit
	if int(resp.ContentLength) > is.httpProxy.Options.ResponseBodySize {
		return OutHttpRequestError(c.Response(), http.StatusOK, model.RequestBodyLimitError)
	}

	// copy header
	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Response().Header().Add(k, v)
		}
	}

	// status code
	c.Response().WriteHeader(resp.StatusCode)

	// copy body
	_, err = io.Copy(c.Response(), resp.Body)
	if err != nil {
		return InflowOutHttpRequestError(c.Response(), c.Request(), http.StatusOK, err)
	}

	return nil
}
