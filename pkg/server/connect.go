package server

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HughNian/nmid/pkg/alert"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/security"
	"github.com/HughNian/nmid/pkg/utils"
)

type Connect struct {
	sync.RWMutex

	closed    uint32
	closeOnce sync.Once

	writeCh chan []byte
	closeCh chan struct{}

	Id     string
	Addr   string
	Ip     string
	Port   string
	Ser    *Server
	Conn   net.Conn
	reader *bufio.Reader
	// buf  bytes.Buffer
	// rw        *bufio.ReadWriter
	ConnType  uint32
	RunWorker *SWorker
	RunClient *SClient

	// isFree uint32

	DataType uint32
	DataLen  uint32
}

type ConnectPool struct {
	sync.Mutex

	TotalNum uint32
	FreeNum  uint32
	Pool     []*Connect
	Free     []*Connect

	CMaps sync.Map
}

func NewConnectPool() *ConnectPool {
	return &ConnectPool{}
}

func (pool *ConnectPool) NewConnect(ser *Server, conn net.Conn) (c *Connect) {
	addr := conn.RemoteAddr().String()
	ip, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil
	}

	//DoWhiteList do whitelist (if use security goup for cloud services you can remove this whitelist)
	if ser.SConfig.WhiteList.Enable && !security.DoWhiteList(ip) {
		go func() {
			zinfo := utils.GetIPZone(ip)
			// logger.Infof("not in whitelist ip %s, ip zone %s", ip, zinfo.Zone)
			alert.SendMarkDownAtAll(alert.DWARNING, "threat ip", fmt.Sprintf("not in whitelist ip %s, ip zone %s", ip, zinfo.Zone))
			ThreatIpCount.Inc(ip, zinfo.Zone, zinfo.Country, zinfo.Prov, zinfo.City, zinfo.Lat, zinfo.Lon)
		}()
		conn.Close()
		return nil
	}

	//DoBlackList do blacklist (if use security goup for cloud services you can remove this blacklist)
	if ser.SConfig.BlackList.Enable && security.DoBlackList(ip) {
		go func() {
			zinfo := utils.GetIPZone(ip)
			// logger.Infof("blacklist ip %s, ip zone %s", ip, zinfo.Zone)
			alert.SendMarkDownAtAll(alert.DWARNING, "threat ip", fmt.Sprintf("blacklist ip %s, ip zone %s", ip, zinfo.Zone))
			ThreatIpCount.Inc(ip, zinfo.Zone, zinfo.Country, zinfo.Prov, zinfo.City, zinfo.Lat, zinfo.Lon)
		}()
		conn.Close()
		return nil
	}

	c = &Connect{
		writeCh: make(chan []byte, 4096),
		closeCh: make(chan struct{}),
	}
	c.Id = utils.GetId() //uuid.Must(uuid.NewRandom()).String()
	c.Addr = addr
	c.Ip = ip
	c.Port = port
	c.Ser = ser
	c.Conn = conn
	c.reader = bufio.NewReaderSize(conn, 32*1024)
	c.ConnType = model.CONN_TYPE_INIT
	c.RunWorker = nil
	c.RunClient = nil

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	go c.writeLoop()

	return c
}

func (c *Connect) closeSignal() {
	if atomic.SwapUint32(&c.closed, 1) != 0 {
		return
	}
	c.closeOnce.Do(func() {
		close(c.closeCh)
	})
}

func (pool *ConnectPool) GetConnect(id string) *Connect {
	pool.Lock()
	item, ok := pool.CMaps.Load(id)
	pool.Unlock()

	if ok {
		return item.(*Connect)
	}

	return nil
}

func (pool *ConnectPool) DelConnect(id string) {
	pool.Lock()
	pool.CMaps.Delete(id)
	pool.Unlock()
}

func (c *Connect) CloseConnect() {
	if c.Conn != nil {
		c.closeSignal()
		c.Conn.Close()
		c.Conn = nil
		c.reader = nil
		c.RunWorker = nil

		if c.RunClient != nil {
			c.RunClient.Timer.Stop()
			c.RunClient = nil
		}
	}
}

func (c *Connect) CloseWorkerConnect() {
	if c.Conn != nil {
		c.closeSignal()
		c.Conn.Close()
		c.Conn = nil
		c.reader = nil
		c.RunWorker.CloseSelfWorker()
		c.RunWorker = nil
	}
}

func (c *Connect) CloseClientConnect() {
	if c.Conn != nil {
		c.closeSignal()
		c.Conn.Close()
		c.Conn = nil
		c.reader = nil
		if c.RunClient != nil {
			c.RunClient.Timer.Stop()
			c.RunClient = nil
		}
	}
}

func (c *Connect) getSWClinet() *SWorker {
	if c.RunWorker == nil {
		c.RunWorker = NewSWorker(c)
	}

	return c.RunWorker
}

func (c *Connect) getSCClient() *SClient {
	if c.RunClient == nil {
		c.RunClient = NewSClient(c)
	}

	//c.Conn.SetDeadline(time.Now().Add(conf.CLIENT_ALIVE_TIME))

	return c.RunClient
}

func (c *Connect) Write(resPack []byte) error {
	if atomic.LoadUint32(&c.closed) != 0 {
		return errors.New("connect closed")
	}
	if c.Conn == nil {
		return errors.New("conn nil")
	}
	select {
	case c.writeCh <- resPack:
		return nil
	default:
		return errors.New("write queue full")
	}
}

func (c *Connect) writeLoop() {
	for {
		select {
		case <-c.closeCh:
			return
		case buf := <-c.writeCh:
			if buf == nil {
				continue
			}
			conn := c.Conn
			if conn == nil {
				c.closeSignal()
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			for len(buf) > 0 {
				n, err := conn.Write(buf)
				if err != nil {
					logger.Error("conn write error", err)
					c.closeSignal()
					_ = conn.Close()
					return
				}
				buf = buf[n:]
			}
		}
	}
}

func (c *Connect) ReadFrame() (connType uint32, dataType uint32, payload []byte, err error) {
	const maxFrameSize = 32 * 1024 * 1024

	if c.Conn == nil {
		return 0, 0, nil, errors.New("conn nil")
	}

	r := io.Reader(c.Conn)
	if c.reader != nil {
		r = c.reader
	}

	header := make([]byte, model.MIN_DATA_SIZE)
	if _, err = io.ReadFull(r, header); err != nil {
		return 0, 0, nil, err
	}

	connType = binary.BigEndian.Uint32(header[:4])
	dataType = binary.BigEndian.Uint32(header[4:8])
	dataLen := int(binary.BigEndian.Uint32(header[8:model.MIN_DATA_SIZE]))

	if connType != model.CONN_TYPE_WORKER && connType != model.CONN_TYPE_CLIENT && connType != model.CONN_TYPE_SERVICE {
		return 0, 0, nil, fmt.Errorf("invalid conn type: %d", connType)
	}
	if dataLen < 0 || dataLen > maxFrameSize {
		return 0, 0, nil, fmt.Errorf("invalid frame size: %d", dataLen)
	}
	if dataLen == 0 {
		return connType, dataType, nil, nil
	}

	payload = make([]byte, dataLen)
	if _, err = io.ReadFull(r, payload); err != nil {
		return 0, 0, nil, err
	}

	return connType, dataType, payload, nil
}

func (c *Connect) DoIO() {
	var err error
	var connType, dataType uint32
	var payload []byte
	var worker *SWorker
	var client *SClient

	for {
		if connType, dataType, payload, err = c.ReadFrame(); err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				if opErr.Temporary() {
					continue
				} else {
					if c.ConnType == model.CONN_TYPE_WORKER {
						workerName := ""
						if c.RunWorker != nil {
							workerName = c.RunWorker.WorkerName
						}
						logger.Errorf("server read worker error conntype:%d, worker ip:%s, worker name:%s, err:%s", c.ConnType, c.Ip, workerName, err.Error())
						alert.SendMarkDownAtAll(alert.DERROR, "worker close", fmt.Sprintf("worker ip: %s, worker name: %s", c.Ip, workerName))
						//do prometheus worker close count
						WorkerCloseCount.Inc(c.Ip)

						c.CloseWorkerConnect()
					}

					if c.ConnType == model.CONN_TYPE_CLIENT {
						c.CloseClientConnect()
					}

					break
				}
			} else if err == io.EOF {
				if c.ConnType == model.CONN_TYPE_WORKER {
					workerName := ""
					if c.RunWorker != nil {
						workerName = c.RunWorker.WorkerName
					}
					logger.Errorf("server read worker error conntype:%d, worker ip:%s, worker name:%s, err:%s", c.ConnType, c.Ip, workerName, err.Error())
					alert.SendMarkDownAtAll(alert.DERROR, "worker close", fmt.Sprintf("worker ip: %s, worker name: %s", c.Ip, workerName))
					//do prometheus worker close count
					WorkerCloseCount.Inc(c.Ip)

					c.Ser.Funcs.DelWorker(c.Id)
				}

				break
			}
			logger.Error("server read error", err)
			c.CloseConnect()
			break
		}

		c.ConnType = connType
		if c.ConnType == model.CONN_TYPE_WORKER {
			worker = c.getSWClinet()
			if worker == nil {
				continue
			}
			worker.Connect.DataType = dataType
			worker.Connect.DataLen = uint32(len(payload))
			worker.RunWorker(payload)
		} else if c.ConnType == model.CONN_TYPE_CLIENT {
			client = c.getSCClient()
			if client == nil {
				continue
			}
			client.Req.DataType = dataType
			client.Req.DataLen = uint32(len(payload))
			client.Req.Data = payload
			client.RunClient()
		}
	}
}
