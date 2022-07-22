package server

import (
	"nmid-v2/pkg/conf"
	"sync"
	"time"

	"github.com/joshbohde/codel"
	"github.com/juju/ratelimit"
)

type SClient struct {
	sync.Mutex

	Timer *time.Timer

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
		Timer:         time.NewTimer(conf.CLIENT_ALIVE_TIME), //todo 后期设置成配置文件
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
		c.Res.DataType = conf.PDT_ERROR
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)

		return
	}
	if c.Req.ParamsLen == 0 || len(c.Req.Params) == 0 {
		c.Res.DataType = conf.PDT_ERROR
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)

		return
	}

	worker := c.Connect.Ser.Funcs.GetBestWorker(c.Req.Handle)
	if worker == nil {
		c.Res.DataType = conf.PDT_CANT_DO
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)

		return
	}

	job := NewJobData(c.Req.Handle, string(c.Req.Params))
	job.Lock()
	job.WorkerId = worker.WorkerId
	job.ClientId = c.ClientId
	job.Unlock()
	job.FuncName = c.Req.Handle
	job.Params = c.Req.Params
	job.ParamsType = c.Req.ParamsType
	job.ParamsHandleType = c.Req.ParamsHandleType

	if ok := worker.Jobs.PushJobData(job); ok {
		worker.Lock()
		worker.JobNum++
		worker.Unlock()
	}

	worker.doWork(job)
}

func (c *SClient) doLimit() {
	c.Res.DataType = conf.PDT_RATELIMIT
	resPack := c.Res.ResEncodePack()
	c.Connect.Write(resPack)
}

//RunClient 此处做限流操作
func (c *SClient) RunClient() {
	if !DoBucketLimiter(c.BucketLimiter) { //令牌桶限流
		c.doLimit()
	} else {
		dataType := c.Req.GetReqDataType()

		switch dataType {
		case conf.PDT_C_DO_JOB:
			{
				c.doJob()
			}
		}
	}
}

//AliveTimeOut 客户端长连接时长限制
func (c *SClient) AliveTimeOut() {
	go func(t *time.Timer) {
		for {
			select {
			case <-t.C:
				c.Connect.CloseConnect()
				t.Reset(conf.CLIENT_ALIVE_TIME)
			}
		}
	}(c.Timer)
}
