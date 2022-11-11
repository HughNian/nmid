package sidecar

import (
	"net"
	"net/http"
	"net/http/httputil"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/utils"
	"time"
)

//HttpProxy server
//do http proxy to the target service, service just a http server

type httpProxy struct {
	Options *model.ProxyServerOption
	Server  *http.Server
	Proxy   *httputil.ReverseProxy
}

func NewHttpProxy(opt *model.ProxyServerOption) (*httpProxy, error) {
	if opt.ReadTimeout.Seconds() <= 0 {
		opt.ReadTimeout = 60 * time.Second
	}
	if opt.WriteTimeout.Seconds() <= 0 {
		opt.WriteTimeout = 60 * time.Second
	}

	proxy := &httpProxy{
		Options: opt,
		Server: &http.Server{
			ReadTimeout:  opt.ReadTimeout,
			WriteTimeout: opt.WriteTimeout,
			IdleTimeout:  opt.IdleTimeout,
			Addr:         opt.BindAddress,
		},
		Proxy: &httputil.ReverseProxy{
			Director: func(request *http.Request) {
				return
			},
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: opt.TargetKeepAlive,
				}).DialContext,
				DisableKeepAlives:   false,
				MaxIdleConnsPerHost: opt.TargetMaxIdleConnsPerHost,
			},
			BufferPool: utils.NewBufferPool(),
		},
	}

	return proxy, nil
}
