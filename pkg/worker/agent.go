package worker

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	"sync"

	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/utils"
)

type Agent struct {
	sync.RWMutex

	net, addr string
	conn      net.Conn
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

func (a *Agent) Read() (data []byte, err error) {
	n := 0
	temp := utils.GetBuffer(model.MIN_DATA_SIZE)
	var buf bytes.Buffer

	// if n, err = a.rw.Read(temp); err != nil {
	// 	return []byte(``), err
	// }

	if n, err = a.conn.Read(temp); err != nil {
		return []byte(``), err
	}

	dataLen := int(binary.BigEndian.Uint32(temp[8:model.MIN_DATA_SIZE]))
	buf.Write(temp[:n])

	for buf.Len() < model.MIN_DATA_SIZE+dataLen {
		tmpcontent := utils.GetBuffer(dataLen)
		// if n, err = a.rw.Read(tmpcontent); err != nil {
		// 	return buf.Bytes(), err
		// }
		if n, err = a.conn.Read(tmpcontent); err != nil {
			return buf.Bytes(), err
		}

		buf.Write(tmpcontent[:n])
	}

	return buf.Bytes(), nil
}

func (a *Agent) Write() (err error) {
	var n int
	buf := a.Req.EncodePack()

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
	var data, leftData []byte
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			if data, err = a.Read(); err != nil {
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

			if len(leftData) > 0 {
				data = append(leftData, data...)
			}

			if len(data) < model.MIN_DATA_SIZE {
				leftData = data
				continue
			}

			if resp, l, err := DecodePack(data); err != nil {
				leftData = data
				continue
			} else if l != len(data) {
				leftData = data
				continue
			} else {
				leftData = nil

				resp.Agent = a
				a.Worker.Resps <- resp
			}
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
