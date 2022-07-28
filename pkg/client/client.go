package client

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"nmid-v2/pkg/model"
	"nmid-v2/pkg/utils"
	"sync"
	"time"
)

//rpc tcp client

type Client struct {
	sync.Mutex

	net, addr string
	conn      net.Conn
	rw        *bufio.ReadWriter

	Req      *Request
	ResQueue chan *Response

	IoTimeOut time.Duration

	ErrHandler   ErrHandler
	RespHandlers *RespHandlerMap
}

func NewClient(network, addr string) (client *Client, err error) {
	client = &Client{
		net:          network,
		addr:         addr,
		Req:          nil,
		ResQueue:     make(chan *Response, model.QUEUE_SIZE),
		IoTimeOut:    model.DEFAULT_TIME_OUT,
		RespHandlers: NewResHandlerMap(),
	}

	err = client.ClientConn()
	if nil != err {
		return nil, err
	}

	go client.ClientRead()

	return client, nil
}

func (c *Client) ClientConn() error {
	var err error

	c.conn, err = net.DialTimeout(c.net, c.addr, model.DIAL_TIME_OUT)
	if err != nil {
		return err
	}
	//if tcpCon, ok := c.conn.(*net.TCPConn); ok {
	//	tcpCon.SetLinger(0)
	//}

	c.rw = bufio.NewReadWriter(bufio.NewReader(c.conn), bufio.NewWriter(c.conn))

	return nil
}

func (c *Client) Write() (err error) {
	var n int
	buf := c.Req.EncodePack()
	for i := 0; i < len(buf); i += n {
		n, err = c.rw.Write(buf)
		if err != nil {
			return err
		}
	}

	return c.rw.Flush()
}

func (c *Client) Read(length int) (data []byte, err error) {
	n := 0
	buf := utils.GetBuffer(length)
	for i := length; i > 0 || len(data) < model.MIN_DATA_SIZE; i -= n {
		if n, err = c.rw.Read(buf); err != nil {
			return
		}
		data = append(data, buf[0:n]...)
		if n < model.MIN_DATA_SIZE {
			break
		}
	}

	return
}

func (c *Client) ClientRead() {
	var data, leftdata []byte
	var err error
	var res *Response
	var resLen int
Loop:
	for c.conn != nil {
		if data, err = c.Read(model.MIN_DATA_SIZE); err != nil {
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
			log.Println("client read error here:" + err.Error())
			c.Close()
			err = c.ClientConn()
			if nil != err {
				break
			}
			c.ResQueue = make(chan *Response, model.QUEUE_SIZE)
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
				c.ResQueue <- res
			}

			data = data[l:]
			if len(data) > 0 {
				continue
			}
			break
		}
	}
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

func (c *Client) ProcessResp() {
	var timer = time.After(c.IoTimeOut)
	select {
	case res := <-c.ResQueue:
		if nil != res {
			switch res.DataType {
			case model.PDT_ERROR:
				c.ErrHandler(res.GetResError())
				return
			case model.PDT_CANT_DO:
				c.ErrHandler(res.GetResError())
				return
			case model.PDT_RATELIMIT:
				c.ErrHandler(res.GetResError())
				return
			case model.PDT_S_RETURN_DATA:
				c.HandlerResp(res)
				return
			}
		}
	case <-timer:
		log.Println("time out")
		c.ErrHandler(model.RESTIMEOUT)
		//c.Close()
		return
	}
}

func (c *Client) SetParamsType(pType uint32) *Client {
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
	c.Lock()
	defer c.Unlock()

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
	if c.conn != nil {
		c.conn.Close()
		close(c.ResQueue)
		c.conn = nil
	}
}
