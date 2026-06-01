package worker

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/utils"
)

type Agent struct {
	sync.RWMutex

	net, addr string
	conn      net.Conn
	reader    *bufio.Reader
	// rw        *bufio.ReadWriter

	Worker   *Worker
	Req      *Request
	Res      *Response
	ctx      context.Context
	cancel   context.CancelFunc
	lastTime int64
}

func NewAgent(net, adrr string, w *Worker) *Agent {
	agent := &Agent{
		net:      net,
		addr:     adrr,
		Worker:   w,
		Req:      NewReq(),
		Res:      NewRes(),
		lastTime: utils.GetNowSecond(),
	}
	agent.ctx, agent.cancel = context.WithCancel(context.Background())

	return agent
}

func (a *Agent) Connect() (err error) {
	a.conn, err = net.DialTimeout(a.net, a.addr, model.DIAL_TIME_OUT)
	if err != nil {
		log.Println("dial error:", err)
		return err
	}
	a.reader = bufio.NewReaderSize(a.conn, 32*1024)

	if tcpCon, ok := a.conn.(*net.TCPConn); ok {
		_ = tcpCon.SetNoDelay(true)
		_ = tcpCon.SetKeepAlive(true)
		_ = tcpCon.SetKeepAlivePeriod(30 * time.Second)
	}
	// a.rw = bufio.NewReadWriter(bufio.NewReader(a.conn), bufio.NewWriter(a.conn))

	go a.Work()

	return nil
}

func (a *Agent) ReConnect() error {
	conn, err := net.DialTimeout(a.net, a.addr, model.DIAL_TIME_OUT)
	if err != nil {
		return err
	}
	a.conn = conn
	a.reader = bufio.NewReaderSize(a.conn, 32*1024)
	// a.rw = bufio.NewReadWriter(bufio.NewReader(a.conn), bufio.NewWriter(a.conn))
	a.lastTime = utils.GetNowSecond()

	return nil
}

func (a *Agent) DelOldFuncMsg(funcName string) {
	a.Req.DelFunctionPack(funcName)
	a.Write()
}

func (a *Agent) ReAddFuncMsg(funcName string) {
	a.Req.AddFunctionPack(funcName)
	a.Write()
}

func (a *Agent) ReSetWorkerName(workerName string) {
	a.Req.SetWorkerName(workerName)
	a.Write()
}

func (a *Agent) ReadFrame() (data []byte, err error) {
	const maxFrameSize = 32 * 1024 * 1024

	if a.conn == nil {
		return nil, fmt.Errorf("conn nil")
	}

	r := io.Reader(a.conn)
	if a.reader != nil {
		r = a.reader
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

	return data, nil
}

func (a *Agent) Write() (err error) {
	var n int
	buf := a.Req.EncodePack()

	if a.conn == nil {
		return fmt.Errorf("conn nil")
	}

	for i := 0; i < len(buf); i += n {
		// if n, err = a.rw.Write(buf[i:]); err != nil {
		// 	return err
		// }
		if n, err = a.conn.Write(buf[i:]); err != nil {
			return err
		}
	}

	// return a.rw.Flush()
	return nil
}

func (a *Agent) Work() {
	var err error
	var data []byte
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			if data, err = a.ReadFrame(); err != nil {
				if opErr, ok := err.(*net.OpError); ok {
					if opErr.Temporary() {
						continue
					} else {
						break
					}
				} else if err == io.EOF {
					break
				}
			}
			resp, _, derr := DecodePack(data)
			if derr != nil || resp == nil {
				continue
			}
			resp.Agent = a
			a.Worker.Resps <- resp
		}
	}
}

func (a *Agent) HeartBeatPing() {
	// logger.Infof("worker heartbeat ping")
	a.Lock()
	a.Req.HeartBeatPack()
	a.Write()
	a.Unlock()
}

func (a *Agent) Grab() {
	a.Lock()
	a.Req.GrabDataPack()
	a.Write()
	a.Unlock()
}

func (a *Agent) Wakeup() {
	a.Lock()
	a.Req.WakeupPack()
	a.Write()
	a.Unlock()

	return
}

func (a *Agent) LimitExceed() {
	a.Lock()
	a.Req.LimitExceedPack()
	a.Write()
	a.Unlock()
}

func (a *Agent) Close() {
	if a.conn != nil {
		a.conn.Close()
		a.conn = nil
		a.cancel()
	}
}
