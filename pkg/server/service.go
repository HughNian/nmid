package server

import (
	"nmid-v2/pkg/conf"
	"sync"
	"time"
)

type Service struct {
	sync.Mutex

	Timer *time.Timer

	Connect *Connect

	Infos map[string]*ServiceInfo
	Req   *Request
	Res   *Response
}

type ServiceInfo struct {
	ServiceId       string
	ServiceName     string
	ServiceHost     string
	ServiceHostName string
	ServiceEnv      string
	ServicePort     uint32
}

func NewService(conn *Connect) *Service {
	if conn == nil {
		return nil
	}

	return &Service{
		Timer:   time.NewTimer(conf.CLIENT_ALIVE_TIME),
		Connect: conn,
		Req:     NewReq(),
		Res:     NewRes(),
	}
}

//DoRegService reg service
func (sc *Service) DoRegService() {

}

//DoOffService log off service
func (sc *Service) DoOffService() {

}

//ConnectNmidRegistry connect nmid registry(register center)
func (sc *Service) ConnectNmidRegistry() {

}

//RunService request register center to reg service
func (sc *Service) RunService() {

}

//AliveTimeOut 客户端长连接时长限制
func (sc *Service) AliveTimeOut() {
	go func(t *time.Timer) {
		for {
			select {
			case <-t.C:
				sc.Connect.CloseConnect()
				t.Reset(conf.CLIENT_ALIVE_TIME)
			}
		}
	}(sc.Timer)
}
