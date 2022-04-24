package server

import (
	"sync"

	"github.com/joshbohde/codel"
	"github.com/juju/ratelimit"
)

type SClient struct {
	sync.Mutex

	ClientId string
	Connect  *Connect

	Req *Request
	Res *Response

	CodelLimiter  *codel.Lock
	BucketLimiter *ratelimit.Bucket
}

func NewSClient(conn *Connect) *SClient {
	if conn == nil {
		return nil
	}

	return &SClient{
		ClientId:      conn.Id,
		Connect:       conn,
		Req:           NewReq(),
		Res:           NewRes(),
		CodelLimiter:  NewCodelLimiter(),
		BucketLimiter: NewBucketLimiter(),
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

//runclient 此处做限流操作
func (c *SClient) RunClient() {
	if !DoBucketLimiter(c.BucketLimiter) { //令牌桶限流
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
