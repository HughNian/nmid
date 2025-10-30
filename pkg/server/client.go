package server

import (
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/model"
)

type SClient struct {
	sync.Mutex

	Timer *time.Timer

	ClientId string
	Connect  *Connect

	Req *Request
	Res *Response

	// CodelLimiter  *codel.Lock
	// BucketLimiter *ratelimit.Bucket
}

func NewSClient(conn *Connect) *SClient {
	if conn == nil {
		return nil
	}

	return &SClient{
		Timer:    time.NewTimer(model.CLIENT_ALIVE_TIME), //todo 后期设置成配置文件
		ClientId: conn.Id,
		Connect:  conn,
		Req:      NewReq(),
		Res:      NewRes(),
		// CodelLimiter:  limiter.NewCodelLimiter(), //todo 由于client大部分是用完即close，所以限流做在worker中
		// BucketLimiter: limiter.NewBucketLimiter(),//todo 由于client大部分是用完即close，所以限流做在worker中
	}
}

func (c *SClient) doJob() {
	c.Req.ReqDecodePack()

	if c.Req.HandleLen == 0 || c.Req.Handle == `` {
		c.Res.DataType = model.PDT_ERROR
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)

		return
	}
	if c.Req.ParamsLen == 0 || len(c.Req.Params) == 0 {
		c.Res.DataType = model.PDT_ERROR
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)

		return
	}

	worker := c.Connect.Ser.Funcs.GetBestWorker(c.Req.Handle)
	if worker == nil {
		c.Res.DataType = model.PDT_CANT_DO
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)

		return
	}

	job := NewJobData(c.Req.Handle, string(c.Req.Params))
	job.Lock()
	job.WorkerId = worker.WorkerId
	job.Client = c.Connect
	job.ClientId = c.ClientId
	job.FuncName = c.Req.Handle
	job.Params = c.Req.Params
	job.ParamsType = c.Req.ParamsType
	job.ParamsHandleType = c.Req.ParamsHandleType
	job.Unlock()

	worker.PushJobToList(job)
	// worker.PushJobToChannel(job)
	worker.doWork(job)

	//do prometheus request count
	requestCount.Inc(worker.WorkerName, c.Req.Handle)
}

func (c *SClient) doLimit() {
	c.Res.DataType = model.PDT_RATELIMIT
	resPack := c.Res.ResEncodePack()
	c.Connect.Write(resPack)
}

// RunClient 此处做限流操作
func (c *SClient) RunClient() {
	// if !limiter.DoBucketLimiter(c.BucketLimiter) { //令牌桶限流
	// 	c.doLimit()
	// } else {
	dataType := c.Req.GetReqDataType()

	switch dataType {
	case model.PDT_C_DO_JOB:
		{
			c.doJob()
		}
	}
	// }
}
