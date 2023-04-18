package server

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/HughNian/nmid/pkg/alert"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/security"
	"github.com/HughNian/nmid/pkg/utils"
)

type Connect struct {
	sync.RWMutex

	Id        string
	Addr      string
	Ip        string
	Port      string
	Ser       *Server
	Conn      net.Conn
	rw        *bufio.ReadWriter
	ConnType  uint32
	RunWorker *SWorker
	RunClient *SClient

	isFree uint32
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
	//DoWhiteList do whitelist
	if ser.SConfig.WhiteList.Enable && !security.DoWhiteList(ip, ser.SConfig.WhiteList) {
		ipzone := utils.GetIPZone(ip)
		logger.Infof("not in whitelist ip %s, ip zone %s", ip, ipzone)
		alert.SendMarkDownAtAll(alert.DWARNING, "threat ip", fmt.Sprintf("not in whitelist ip %s, ip zone %s", ip, ipzone))
		conn.Close()
		return nil
	}
	//DoBlackList do blacklist
	if ser.SConfig.BlackList.Enable && security.DoBlackList(ip, ser.SConfig.BlackList) {
		ipzone := utils.GetIPZone(ip)
		logger.Infof("blacklist ip %s, ip zone %s", ip, ipzone)
		alert.SendMarkDownAtAll(alert.DWARNING, "threat ip", fmt.Sprintf("blacklist ip %s, ip zone %s", ip, ipzone))
		conn.Close()
		return nil
	}

	c = &Connect{}
	c.Id = utils.GetId() //uuid.Must(uuid.NewRandom()).String()
	c.Addr = addr
	c.Ip = ip
	c.Port = port
	c.Ser = ser
	pool.Lock()
	c.Conn = conn
	c.rw = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	pool.Unlock()
	c.ConnType = model.CONN_TYPE_INIT
	c.RunWorker = nil
	c.RunClient = nil

	return c
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
		c.Conn.Close()
		c.Conn = nil
		c.RunWorker = nil

		c.RunClient.Timer.Stop()
		c.RunClient = nil
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

	c.RunClient.AliveTimeOut()
	//c.Conn.SetDeadline(time.Now().Add(conf.CLIENT_ALIVE_TIME))

	return c.RunClient
}

func (c *Connect) Write(resPack []byte) {
	var n int
	var err error
	if c.ConnType == model.CONN_TYPE_WORKER {
		worker := c.RunWorker
		for i := 0; i < len(resPack); i += n {
			n, err = worker.Connect.rw.Write(resPack[i:])
			if err != nil {
				logger.Error("write err %s", err.Error())
				return
			}
		}

		worker.Connect.rw.Flush()

		// _, err = worker.Connect.Conn.Write(resPack[:])
		// if err != nil {
		// 	logger.Info("worker write err", err.Error())
		// 	return
		// }
	} else if c.ConnType == model.CONN_TYPE_CLIENT {
		client := c.RunClient

		for i := 0; i < len(resPack); i += n {
			n, err = client.Connect.rw.Write(resPack[i:])
			if err != nil {
				logger.Info("client write err", err.Error())
				return
			}
		}

		client.Connect.rw.Flush()

		// n, err = client.Connect.Conn.Write(resPack[:])
		// if err != nil {
		// 	logger.Info("client write err", err.Error())
		// 	return
		// }
	}
}

func (c *Connect) Read(size int) (data []byte, err error) {
	n := 0
	var buf bytes.Buffer
	var connType, dataType uint32
	var dataLen int
	tmp := utils.GetBuffer(size)

	if n, err = c.Conn.Read(tmp); err != nil {
		if c.ConnType == model.CONN_TYPE_WORKER {
			logger.Errorf("server read worker error conntype:%d, worker ip:%s, err:%s", c.ConnType, c.Ip, err.Error())
		}
		return []byte(``), err
	}

	//读取数据头
	if n >= model.MIN_DATA_SIZE {
		connType = uint32(binary.BigEndian.Uint32(tmp[:4]))
		dataType = uint32(binary.BigEndian.Uint32(tmp[4:8]))
		dataLen = int(binary.BigEndian.Uint32(tmp[8:model.MIN_DATA_SIZE]))

		if connType != model.CONN_TYPE_WORKER &&
			connType != model.CONN_TYPE_CLIENT &&
			connType != model.CONN_TYPE_SERVICE {
			return []byte(``), nil
		}

		c.ConnType = connType
		if c.ConnType == model.CONN_TYPE_WORKER {
			worker := c.getSWClinet()
			worker.Req.DataType = dataType
			worker.Req.DataLen = uint32(dataLen)
		} else if c.ConnType == model.CONN_TYPE_CLIENT {
			client := c.getSCClient()
			client.Req.DataType = dataType
			client.Req.DataLen = uint32(dataLen)
		}

		buf.Write(tmp[:n])
	} else {
		buf.Write(tmp[:n])

		return buf.Bytes(), nil
	}

	//读取所有内容
	for buf.Len() < dataLen+model.MIN_DATA_SIZE {
		tmpcontent := utils.GetBuffer(dataLen)
		if n, err = c.Conn.Read(tmpcontent); err != nil {
			logger.Error("read content error")
			return buf.Bytes(), err
		}

		buf.Write(tmpcontent[:n])
	}

	return buf.Bytes(), err
}

func (c *Connect) DoIO() {
	var err error
	var data, content []byte
	var rsize = model.MIN_DATA_SIZE
	var worker *SWorker
	var client *SClient

	for {
		if data, err = c.Read(rsize); err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				if opErr.Temporary() {
					continue
				} else {
					//c.Ser.Cpool.DelConnect(c.Id)
					break
				}
			} else if err == io.EOF {
				if c.ConnType == model.CONN_TYPE_WORKER {
					c.Ser.Funcs.DelWorker(c.Id)
				}
				//c.Ser.Cpool.DelConnect(c.Id)
				break
			}
		}

		if c.ConnType == model.CONN_TYPE_WORKER {
			worker = c.RunWorker

			allLen := uint32(len(data))
			if worker.Req.DataLen > allLen {
				continue
			}

			content = make([]byte, worker.Req.DataLen)
			copy(content, data[model.MIN_DATA_SIZE:allLen])
			clen := uint32(len(content))
			if worker.Req.DataLen == clen {
				worker.Req.Data = content
				worker.RunWorker()
			}
		} else if c.ConnType == model.CONN_TYPE_CLIENT {
			client = c.RunClient

			allLen := uint32(len(data))
			if client.Req.DataLen > allLen {
				continue
			}

			content = make([]byte, client.Req.DataLen)
			copy(content, data[model.MIN_DATA_SIZE:allLen])
			clen := uint32(len(content))
			if client.Req.DataLen == clen {
				client.Req.Data = content
				client.RunClient()
			}
		} else {
			continue
		}
	}
}
