package server

import (
	"context"
	"sync"
	"time"

	"github.com/joshbohde/codel"
)

type SClient struct {
	sync.Mutex

	ClientId string
	Connect  *Connect

	Req *Request
	Res *Response
}

func NewSClient(conn *Connect) *SClient {
	if conn == nil {
		return nil
	}

	return &SClient{
		ClientId: conn.Id,
		Connect:  conn,
		Req:      NewReq(),
		Res:      NewRes(),
	}
}

func (c *SClient) doJob() {
	c.Req.ReqDecodePack()

	// fmt.Println("######Client Req-", c.Req.DataType)

	if c.Req.HandleLen == 0 || c.Req.Handle == `` {
		c.Res.DataType = PDT_ERROR
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)

		return
	}
	if c.Req.ParamsLen == 0 || len(c.Req.Params) == 0 {
		c.Res.DataType = PDT_ERROR
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)

		return
	}

	worker := c.Connect.Ser.Funcs.GetBestWorker(c.Req.Handle)
	if worker == nil {
		c.Res.DataType = PDT_CANT_DO
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)

		return
	}

	job := NewJobData(c.Req.Handle, string(c.Req.Params))
	job.WorkerId = worker.WorkerId
	job.ClientId = c.ClientId
	job.FuncName = c.Req.Handle
	job.Params = c.Req.Params
	if IsMulParams(job.Params) {
		job.ParamsType = PARAMS_TYPE_MUL
	} else {
		job.ParamsType = PARAMS_TYPE_ONE
	}

	if ok := worker.Jobs.PushJobData(job); ok {
		worker.Lock()
		worker.JobNum++
		worker.Unlock()
	}

	worker.doWork()
}

func (c *SClient) doLimit() {
	c.Res.DataType = PDT_RATELIMIT
	resPack := c.Res.ResEncodePack()
	c.Connect.Write(resPack)
}

//codel限流
func codelLimiter() bool {
	c := codel.New(codel.Options{
		// The maximum number of pending acquires
		MaxPending: 100,
		// The maximum number of concurrent acquires
		MaxOutstanding: 10,
		// The target latency to wait for an acquire.
		// Acquires that take longer than this can fail.
		TargetLatency: 5 * time.Millisecond,
	})

	// Attempt to acquire the lock.
	err := c.Acquire(context.Background())

	// if err is not nil, acquisition failed.
	if err != nil {
		return false
	}

	// If acquisition succeeded, we need to release it.
	defer c.Release()

	return true
}

func (c *SClient) RunClient() {
	if !codelLimiter() {
		c.doLimit()
	} else {
		dataType := c.Req.GetReqDataType()

		switch dataType {
		case PDT_C_DO_JOB:
			{
				go c.doJob()
			}
		}
	}
}
