package service

import (
	"bufio"
	"net"
	"net/http"
	"sync"
	"time"
)

//sidecar client

type (
	Service struct {
		sync.Mutex

		net, addr string
		conn      net.Conn
		rw        *bufio.ReadWriter
		scClient  *http.Client

		SInfo *ServiceInfo

		IoTimeOut time.Duration
	}

	ServiceInfo struct {
		ServiceId  string
		InFlowUrl  string
		OutFlowUrl string
		Instance   *Instance
	}

	Instance struct {
		Region      string            `json:"region"`
		Zone        string            `json:"zone"`
		Env         string            `json:"env"`
		ServiceId   string            `json:"serviceId"`
		ServiceName string            `json:"servicename"`
		HostName    string            `json:"hostname"`
		Addrs       []string          `json:"addrs"`
		Version     string            `json:"version"`
		Metadata    map[string]string `json:"metadata"`
	}
)

func (sc *Service) ScHTTPClient() *http.Client {
	if sc.scClient == nil {
		sc.scClient = &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 60 * time.Second,
				}).DialContext,
				MaxIdleConns:          500,
				IdleConnTimeout:       60 * time.Second,
				ExpectContinueTimeout: 30 * time.Second,
				MaxIdleConnsPerHost:   100,
			},
		}
	}

	return sc.scClient
}

func NewService(network, addr string) (service *Service, err error) {
	return
}

//RegService register service 服务注册
func (sc *Service) RegService() (ret bool, err error) {
	return
}

//OffService logoff service 服务下线
func (sc *Service) OffService() (ret bool, err error) {
	return
}

//CallService call target service, use sidecar outflow addr
func (sc *Service) CallService() {

}
