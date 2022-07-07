package service

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Service struct {
	sync.Mutex

	ServiceId   string
	ServiceName string
	ServiceHost string
	ServicePort uint32

	net, addr string
	conn      net.Conn
	rw        *bufio.ReadWriter

	Req      *Request
	ResQueue chan *Response

	IoTimeOut time.Duration
}

func NewService(network, addr string) (service *Service, err error) {
	service = &Service{
		net:       network,
		addr:      addr,
		Req:       nil,
		ResQueue:  make(chan *Response, QUEUE_SIZE),
		IoTimeOut: DEFAULT_TIME_OUT,
	}
	service.conn, err = net.DialTimeout(service.net, service.addr, DIAL_TIME_OUT)
	if err != nil {
		return nil, err
	}

	service.rw = bufio.NewReadWriter(bufio.NewReader(service.conn), bufio.NewWriter(service.conn))

	go service.ServiceRead()

	return service, nil
}

func (sc *Service) SetServiceInfo(ServiceName, ServiceHost string, ServicePort uint32) *Service {
	sc.ServiceName = ServiceName
	sc.ServiceHost = ServiceHost
	sc.ServicePort = ServicePort
	sc.ServiceId = GenServiceId(ServiceName, ServiceHost)
	return sc
}

func (sc *Service) GetServiceId() string {
	return sc.ServiceId
}

func (sc *Service) Write() (err error) {
	var n int
	buf := sc.Req.EncodePack()
	for i := 0; i < len(buf); i += n {
		n, err = sc.rw.Write(buf)
		if err != nil {
			return err
		}
	}

	return sc.rw.Flush()
}

func (sc *Service) Read(length int) (data []byte, err error) {
	n := 0
	buf := GetBuffer(length)
	for i := length; i > 0 || len(data) < MIN_DATA_SIZE; i -= n {
		if n, err = sc.rw.Read(buf); err != nil {
			return
		}
		data = append(data, buf[0:n]...)
		if n < MIN_DATA_SIZE {
			break
		}
	}

	return
}

func (sc *Service) ServiceRead() {
	var data, leftdata []byte
	var err error
	var res *Response
	var resLen int
Loop:
	for sc.conn != nil {
		if data, err = sc.Read(MIN_DATA_SIZE); err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				if opErr.Timeout() {
					log.Println(err)
				}
				if opErr.Temporary() {
					continue
				}
				break
			}

			//服务端断开
			if err == io.EOF {
				//c.ErrHandler(err)
			}

			//断开重连
			log.Println("service read error here:" + err.Error())
			sc.Close()
			sc.conn, err = net.DialTimeout(sc.net, sc.addr, DIAL_TIME_OUT)
			if err != nil {
				break
			}
			sc.rw = bufio.NewReadWriter(bufio.NewReader(sc.conn), bufio.NewWriter(sc.conn))
			sc.ResQueue = make(chan *Response, QUEUE_SIZE)
			continue
		}

		if len(leftdata) > 0 {
			data = append(leftdata, data...)
			leftdata = nil
		}

		for {
			l := len(data)
			if l < MIN_DATA_SIZE {
				leftdata = data
				continue Loop
			}

			if len(leftdata) == 0 {
				connType := GetConnType(data)
				if connType != CONN_TYPE_SERVER {
					log.Println("read conn type error")
					break
				}
			}

			if res, resLen, err = DecodePack(data); err != nil {
				leftdata = data[:resLen]
				continue Loop
			} else {
				sc.ResQueue <- res
			}

			data = data[l:]
			if len(data) > 0 {
				continue
			}
			break
		}
	}
}

func (sc *Service) ProcessResp() bool {
	select {
	case res := <-sc.ResQueue:
		if nil != res {
			switch res.DataType {
			case PDT_ERROR:
				log.Println("pdt error")
				return false
			case PDT_RATELIMIT:
				log.Println("pdt rateLimit")
				return false
			case PDT_S_REG_SERVICE_OK:
				return true
			}
		}
	case <-time.After(sc.IoTimeOut):
		log.Println("time out")
		return false
	}

	return true
}

func (sc *Service) Close() {
	if sc.conn != nil {
		sc.conn.Close()
		close(sc.ResQueue)
		sc.conn = nil
	}
}

//RegService register service 服务注册
func (sc *Service) RegService() (ret bool, err error) {
	sc.Lock()
	defer sc.Unlock()

	if sc.conn == nil {
		return false, fmt.Errorf("conn fail")
	}
	if len(sc.ServiceName) == 0 {
		return false, fmt.Errorf("service name empty")
	}
	if len(sc.ServiceHost) == 0 {
		return false, fmt.Errorf("service host empty")
	}
	if sc.ServicePort == 0 {
		return false, fmt.Errorf("service port err")
	}

	NewReq(ScInfo{
		ServiceId:   sc.ServiceId,
		ServiceName: sc.ServiceName,
		ServiceHost: sc.ServiceHost,
		ServicePort: sc.ServicePort,
	}).ServiceInfoPack(PDT_SC_REG_SERVICE)
	if err = sc.Write(); err != nil {
		return false, err
	}

	return sc.ProcessResp(), nil
}

//OffService logoff service 服务下线
func (sc *Service) OffService() (ret bool, err error) {
	sc.Lock()
	defer sc.Unlock()

	if sc.conn == nil {
		return false, fmt.Errorf("conn fail")
	}
	if len(sc.ServiceName) == 0 {
		return false, fmt.Errorf("service name empty")
	}
	if len(sc.ServiceHost) == 0 {
		return false, fmt.Errorf("service host empty")
	}
	if sc.ServicePort == 0 {
		return false, fmt.Errorf("service port err")
	}

	NewReq(ScInfo{
		ServiceId:   sc.ServiceId,
		ServiceName: sc.ServiceName,
		ServiceHost: sc.ServiceHost,
		ServicePort: sc.ServicePort,
	}).ServiceInfoPack(PDT_SC_OFF_SERVICE)
	if err = sc.Write(); err != nil {
		return false, err
	}

	return sc.ProcessResp(), nil
}
