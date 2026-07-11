package client

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
)

//rpc tcp client

type Client struct {
	sync.Mutex

	net    string
	Addr   string
	conn   net.Conn
	reader *bufio.Reader

	Req      *Request
	ResQueue chan *Response

	IoTimeOut time.Duration

	ErrHandler   ErrHandler
	RespHandlers *RespHandlerMap
}

func NewClient(network, addr string) (client *Client) {
	client = &Client{
		net:          network,
		Addr:         addr,
		Req:          nil,
		ResQueue:     make(chan *Response),
		IoTimeOut:    model.DEFAULT_TIME_OUT,
		RespHandlers: NewResHandlerMap(),
	}

	return client
}

func (c *Client) SetIoTimeOut(t time.Duration) *Client {
	c.IoTimeOut = t
	return c
}

func (c *Client) ClientConn() error {
	c.Lock()
	defer c.Unlock()

	var err error

	c.conn, err = net.DialTimeout(c.net, c.Addr, model.DIAL_TIME_OUT)
	if err != nil {
		return err
	}
	c.reader = bufio.NewReaderSize(c.conn, 32*1024)

	if tcpCon, ok := c.conn.(*net.TCPConn); ok {
		_ = tcpCon.SetNoDelay(true)
		_ = tcpCon.SetKeepAlive(true)
		_ = tcpCon.SetKeepAlivePeriod(30 * time.Second)
	}

	return nil
}

func (c *Client) Start() (client *Client, err error) {
	err = c.ClientConn()
	if nil != err {
		return
	}

	go c.ClientRead()

	return c, err
}

func (c *Client) Write() (err error) {
	c.Lock()
	defer c.Unlock()

	if c.conn == nil {
		return errors.New("conn nil")
	}

	var n int
	buf := c.Req.EncodePack()
	for i := 0; i < len(buf); i += n {
		n, err = c.conn.Write(buf[i:])
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) ReadFrame() (data []byte, err error) {
	const maxFrameSize = 32 * 1024 * 1024

	if c.conn == nil {
		return nil, errors.New("conn nil")
	}

	r := io.Reader(c.conn)
	if c.reader != nil {
		r = c.reader
	}

	header := make([]byte, model.MIN_DATA_SIZE)
	if _, err = io.ReadFull(r, header); err != nil {
		return nil, err
	}

	connType := binary.BigEndian.Uint32(header[:4])
	if connType != model.CONN_TYPE_SERVER {
		return nil, fmt.Errorf("invalid conn type: %d", connType)
	}
	contentLen := int(binary.BigEndian.Uint32(header[8:model.MIN_DATA_SIZE]))
	if contentLen < 0 || contentLen > maxFrameSize {
		return nil, fmt.Errorf("invalid frame size: %d", contentLen)
	}

	data = make([]byte, model.MIN_DATA_SIZE+contentLen)
	copy(data[:model.MIN_DATA_SIZE], header)
	if contentLen > 0 {
		if _, err = io.ReadFull(r, data[model.MIN_DATA_SIZE:]); err != nil {
			return nil, err
		}
	}

	return
}

func (c *Client) ClientRead() {
	var data []byte
	var err error
	var res *Response

	for c.conn != nil {
		if data, err = c.ReadFrame(); err != nil {
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
			logger.Info("client read error here:" + err.Error())
			c.Close()
			err = c.ClientConn()
			if nil != err {
				break
			}
			c.ResQueue = make(chan *Response)
			continue
		}

		if res, _, err = DecodePack(data); err != nil {
			continue
		}
		c.ResQueue <- res
	}
}

func (c *Client) resetReader() {
	if c.conn == nil {
		c.reader = nil
		return
	}
	c.reader = bufio.NewReaderSize(c.conn, 32*1024)
}

func (c *Client) HandlerResp(resp *Response) {
	if resp == nil {
		return
	}
	if len(resp.Handle) == 0 || resp.HandleLen == 0 {
		return
	}

	key := resp.Handle
	if handler, exist := c.RespHandlers.GetResHandlerMap(key); exist {
		handler(resp)
		c.RespHandlers.DelResHandlerMap(key)
		return
	}
}

func (c *Client) safeErrHandler(err error) {
	if c.ErrHandler != nil {
		c.ErrHandler(err)
	}
}

func (c *Client) ProcessResp() {
	var timer = time.After(c.IoTimeOut)
	select {
	case res := <-c.ResQueue:
		if nil != res {
			switch res.DataType {
			case model.PDT_ERROR:
				c.safeErrHandler(res.GetResError())
				return
			case model.PDT_CANT_DO:
				c.safeErrHandler(res.GetResError())
				return
			case model.PDT_RATELIMIT:
				c.safeErrHandler(res.GetResError())
				return
			case model.PDT_S_RETURN_DATA:
				c.HandlerResp(res)
				return
			}
		}
	case <-timer:
		fmt.Println("time out here")
		c.safeErrHandler(model.RESTIMEOUT)
		return
	}
}

func (c *Client) SetParamsType(pType uint32) *Client {
	if nil == c {
		return c
	}

	if pType != model.PARAMS_TYPE_MSGPACK && pType != model.PARAMS_TYPE_JSON {
		log.Println("set params type value error not in msgpack or json")
		return c
	}

	if c.Req == nil {
		c.Req = NewReq()
	}
	c.Req.ParamsType = pType
	return c
}

func (c *Client) SetParamsHandle(hType uint32) *Client {
	if hType != model.PARAMS_HANDLE_TYPE_ENCODE && hType != model.PARAMS_HANDLE_TYPE_ORIGINAL {
		log.Println("set params handle type value error not in encode or original")
		return c
	}

	if c.Req == nil {
		c.Req = NewReq()
	}
	c.Req.ParamsHandleType = hType
	return c
}

func (c *Client) Do(funcName string, params []byte, callback RespHandler) (err error) {
	if c.conn == nil {
		return fmt.Errorf("conn fail")
	}

	c.RespHandlers.PutResHandlerMap(funcName, callback)

	if c.Req == nil {
		c.Req = NewReq()
	}
	c.Req.ContentPack(model.PDT_C_DO_JOB, funcName, params)
	if err = c.Write(); err != nil {
		return err
	}

	c.ProcessResp()

	return nil
}

func (c *Client) Close() {
	if nil == c {
		return
	}

	c.Lock()
	defer c.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		close(c.ResQueue)
		c.RespHandlers.holder = nil
		c.RespHandlers = nil
		c = nil
	}
}
