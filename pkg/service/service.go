package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"nmid-v2/pkg/model"
	"nmid-v2/pkg/utils"
	"sync"
	"time"
)

type (
	Service struct {
		sync.Mutex

		net, addr string
		conn      net.Conn
		rw        *bufio.ReadWriter

		SInfo *ServiceInfo

		Req      *Request
		ResQueue chan *Response

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

func NewService(network, addr string) (service *Service, err error) {
	service = &Service{
		net:       network,
		addr:      addr,
		SInfo:     &ServiceInfo{},
		Req:       nil,
		ResQueue:  make(chan *Response, model.QUEUE_SIZE),
		IoTimeOut: model.DEFAULT_TIME_OUT,
	}
	service.conn, err = net.DialTimeout(service.net, service.addr, model.DIAL_TIME_OUT)
	if err != nil {
		return nil, err
	}

	service.rw = bufio.NewReadWriter(bufio.NewReader(service.conn), bufio.NewWriter(service.conn))

	go service.ServiceRead()

	return service, nil
}

func (sc *Service) SetServiceInfo(inflowUrl, outflowUrl string, instance []byte) *Service {
	if len(inflowUrl) == 0 {
		fmt.Errorf("inflowUrl empty")
		return nil
	}

	sc.SInfo.InFlowUrl = inflowUrl
	sc.SInfo.OutFlowUrl = outflowUrl
	ins := Instance{}
	err := json.Unmarshal(instance, &ins)
	if nil != err {
		log.Fatalln("instance info error", err)
		return nil
	}

	if len(ins.ServiceName) == 0 {
		fmt.Errorf("service name empty")
		return nil
	}
	if len(ins.HostName) == 0 {
		fmt.Errorf("host name empty")
		return nil
	}
	if len(ins.Addrs) == 0 {
		fmt.Errorf("addrs empty")
		return nil
	}
	if len(ins.Metadata) == 0 {
		fmt.Errorf("metadata empty")
		return nil
	}

	sc.SInfo.ServiceId = utils.GenServiceId(ins.ServiceName)
	ins.ServiceId = sc.SInfo.ServiceId
	sc.SInfo.Instance = &ins

	return sc
}

func (sc *Service) GetServiceId() string {
	return sc.SInfo.ServiceId
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
	buf := utils.GetBuffer(length)
	for i := length; i > 0 || len(data) < model.MIN_DATA_SIZE; i -= n {
		if n, err = sc.rw.Read(buf); err != nil {
			return
		}
		data = append(data, buf[0:n]...)
		if n < model.MIN_DATA_SIZE {
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
		if data, err = sc.Read(model.MIN_DATA_SIZE); err != nil {
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
			sc.conn, err = net.DialTimeout(sc.net, sc.addr, model.DIAL_TIME_OUT)
			if err != nil {
				break
			}
			sc.rw = bufio.NewReadWriter(bufio.NewReader(sc.conn), bufio.NewWriter(sc.conn))
			sc.ResQueue = make(chan *Response, model.QUEUE_SIZE)
			continue
		}

		if len(leftdata) > 0 {
			data = append(leftdata, data...)
			leftdata = nil
		}

		for {
			l := len(data)
			if l < model.MIN_DATA_SIZE {
				leftdata = data
				continue Loop
			}

			if len(leftdata) == 0 {
				connType := GetConnType(data)
				if connType != model.CONN_TYPE_SERVER {
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
			case model.PDT_ERROR:
				log.Println("pdt error")
				return false
			case model.PDT_RATELIMIT:
				log.Println("pdt rateLimit")
				return false
			case model.PDT_S_REG_SERVICE_OK:
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

	NewReq(sc.SInfo).ServiceInfoPack(model.PDT_SC_REG_SERVICE)
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

	NewReq(sc.SInfo).ServiceInfoPack(model.PDT_SC_OFF_SERVICE)
	if err = sc.Write(); err != nil {
		return false, err
	}

	return sc.ProcessResp(), nil
}
