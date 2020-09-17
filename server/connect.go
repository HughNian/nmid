package server

import (
	"sync"
	"bufio"
	"net"
	"log"
	"bytes"
	"encoding/binary"
	"io"
	"fmt"
	"strings"
)

type Connect struct {
	sync.Mutex

	Id        string
	Addr      string
	Ip        string
	Ser       *Server
	Conn      net.Conn
	rw        *bufio.ReadWriter
	ConnType  uint32
	RunWorker *SWorker
	RunClient *SClient

	isFree    uint32
}

type ConnectPool struct {
	sync.Mutex

	TotalNum uint32
	FreeNum  uint32
	Pool     []*Connect
	Free	 []*Connect
}

func NewConnectPool() *ConnectPool {
	return &ConnectPool {
		TotalNum:0,
		FreeNum :0,
		Pool:make([]*Connect, MAX_POOL_SIZE),
		Free:make([]*Connect, 0),
	}
}

func (pool *ConnectPool) NewConnect(ser *Server, conn net.Conn) (c *Connect) {
	addr := conn.RemoteAddr().String()
	addrArr := strings.Split(addr, ":")
	ip := ""
	if len(addrArr) > 0 {
		ip = addrArr[0]

		if !AddrAllow(ip) {
			conn.Close()
			return nil
		}
	}

	if pool.FreeNum > 0 {
		fc := pool.Free[0]
		if fc == nil {
			log.Println("free connect nil")
			return
		}
		if fc.isFree != 1 {
			log.Println("connect not free")
			return
		}
		c = fc
		pool.Free = pool.Free[1:] //剔除第一个元素
		pool.FreeNum--
	} else {
		c = new(Connect)
	}

	c.Id   = GetId()
	c.Addr = addr
	c.Ip   = ip
	c.Ser  = ser
	c.Conn = conn
	c.rw   = bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	c.ConnType = CONN_TYPE_INIT
	c.RunWorker = nil
	c.RunClient = nil
	c.isFree = 0

	if pool.TotalNum < MAX_POOL_SIZE {
		pool.Pool[pool.TotalNum] = c
	} else {
		pool.Pool = append(pool.Pool, c)
	}
	pool.TotalNum++

	return c
}

func (pool *ConnectPool) GetConnect(id string) *Connect {
	pool.Lock()
	defer pool.Unlock()

	for _, c := range pool.Pool {
		if c != nil && c.Id == id {
			return c
		}
	}

	return nil
}

func (pool *ConnectPool) DelConnect(id string) {
	if pool.TotalNum == 0 {
		return
	}

	var fc *Connect
	for _, c := range pool.Pool {
		if c != nil && c.Id == id {
			fc = c
			break
		} else {
			fc = nil
		}
	}

	if nil == fc {
		return
	}

	pool.Lock()

	fc.isFree = 1
	fc.CloseConnect()
	pool.Free = append(pool.Free, fc)

	pool.FreeNum++
	pool.TotalNum--

	pool.Unlock()
}

func (c *Connect) CloseConnect() {
	c.Lock()
	defer c.Unlock()

	if c.Conn != nil {
		fmt.Println("here hahaha")
		c.Conn.Close()
		c.Conn = nil
		c.RunWorker = nil
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

	return c.RunClient
}

func (c *Connect) Write(resPack []byte) {
	var n int
	var err error
	fmt.Println("#######ConnType-", c.ConnType)
	if c.ConnType == CONN_TYPE_WORKER {
		dataType := uint32(binary.BigEndian.Uint32(resPack[4:8]))
		fmt.Println("######Worker write dataType-", dataType)

		/*
		n, err = c.Conn.Write(resPack)
		fmt.Println("######Worker write n-", n)
		if err != nil {
			return
		}
		c.rw.Flush()
		*/

		worker := c.RunWorker
		for i := 0; i < len(resPack); i += n {
			//n, err = worker.Connect.Conn.Write(resPack[i:])
			n, err = worker.Connect.rw.Write(resPack[i:])
			//n, err = c.rw.Write(resPack[i:])
			if err != nil {
				return
			}
		}

		worker.Connect.rw.Flush()
	} else if c.ConnType == CONN_TYPE_CLIENT {
		connType := uint32(binary.BigEndian.Uint32(resPack[:4]))
		fmt.Println("######Client write connType-", connType)
		dataType := uint32(binary.BigEndian.Uint32(resPack[4:8]))
		fmt.Println("######Client write dataType-", dataType)

		if len(resPack) == 0 {
			log.Println("resPack nil")
			return
		}

		/*
		//n, err = c.rw.Write(resPack)
		n, err = c.Conn.Write(resPack)
		fmt.Println("######Client write n-", n)
		if err != nil {
			return
		}
		//c.rw.Flush()
		*/

		client := c.RunClient
		fmt.Println("######client-", client)
		fmt.Println("######c-", c)
		for i := 0; i < len(resPack); i += n {
			n, err = c.rw.Write(resPack[i:])
			if err != nil {
				return
			}
		}

		c.rw.Flush()
	}
}

func (c *Connect) Read(size int) (data []byte, err error) {
	n := 0
	var buf bytes.Buffer
	var connType, dataType uint32
	var dataLen int
	tmp := GetBuffer(size)

	if n, err = c.rw.Read(tmp); err != nil {
		log.Println("server read error", c.Ip, err)
		return []byte(``), err
	}

	//读取数据头
	if n >= MIN_DATA_SIZE {
		connType = uint32(binary.BigEndian.Uint32(tmp[:4]))
		dataType = uint32(binary.BigEndian.Uint32(tmp[4:8]))
		dataLen  = int(binary.BigEndian.Uint32(tmp[8:MIN_DATA_SIZE]))

		if connType != CONN_TYPE_WORKER && connType != CONN_TYPE_CLIENT {
			//log.Println("connect type error")
			return []byte(``), nil
		}
		c.ConnType = connType
		if c.ConnType == CONN_TYPE_WORKER {
			//fmt.Println("###### Worker read")

			worker := c.getSWClinet()
			worker.Req.DataType = dataType
			worker.Req.DataLen  = uint32(dataLen)
		} else if c.ConnType == CONN_TYPE_CLIENT {
			//fmt.Println("###### Client read")

			client := c.getSCClient()
			client.Req.DataType = dataType
			client.Req.DataLen  = uint32(dataLen)
		}

		buf.Write(tmp[:n])
	} else {
		buf.Write(tmp[:n])

		return buf.Bytes(), nil
	}

	//读取所有内容
	for buf.Len() < dataLen + MIN_DATA_SIZE {
		tmpcontent := GetBuffer(dataLen)
		if n, err = c.rw.Read(tmpcontent); err != nil {
			log.Println("read content error")
			return buf.Bytes(), err
		}

		buf.Write(tmpcontent[:n])
	}

	fmt.Println("######server read data", buf.Bytes())

	return buf.Bytes(), err
}

func (c *Connect) DoIO() {
	var err error
	var data, content []byte
	var rsize = MIN_DATA_SIZE
	var worker *SWorker
	var client *SClient

	for {
		if data, err = c.Read(rsize); err != nil {
			if opErr, ok := err.(*net.OpError); ok {
				if opErr.Temporary() {
					continue
				} else {
					c.Ser.Cpool.DelConnect(c.Id)
					break
				}
			} else if err == io.EOF {
				fmt.Println("read eof here")
				if c.ConnType == CONN_TYPE_WORKER {
					c.Ser.Funcs.DelWorker(c.Id)
				}
				fmt.Println("client close here")
				c.Ser.Cpool.DelConnect(c.Id)
				break
			}
		}

		if c.ConnType == CONN_TYPE_WORKER {
			worker = c.RunWorker

			allLen := uint32(len(data))
			if worker.Req.DataLen > allLen {
				continue
			}

			content = make([]byte, worker.Req.DataLen)
			copy(content, data[MIN_DATA_SIZE:allLen])
			clen := uint32(len(content))
			if worker.Req.DataLen == clen {
				worker.Req.Data = content
				worker.RunWorker()
			}
		} else if c.ConnType == CONN_TYPE_CLIENT {
			client = c.RunClient

			allLen := uint32(len(data))
			if client.Req.DataLen > allLen {
				continue
			}

			content = make([]byte, client.Req.DataLen)
			copy(content, data[MIN_DATA_SIZE:allLen])
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