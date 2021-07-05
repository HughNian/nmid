package server

import (
	"sync"
	"fmt"
)

type SClient struct {
	sync.Mutex

	ClientId string
	Connect  *Connect

	Req      *Request
	Res      *Response
}

func NewSClient(conn *Connect) *SClient {
	if conn == nil {
		return nil
	}

	return &SClient {
		ClientId : conn.Id,
		Connect  : conn,
		Req      : NewReq(),
		Res      : NewRes(),
	}
}

func (c *SClient) doJob() {
	c.Lock()
	defer c.Unlock()

	c.Req.ReqDecodePack()

	fmt.Println("######Client Req-", c.Req.DataType)

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

	job := NewJob(c.Req.Handle, string(c.Req.Params))
	job.WorkerId = worker.WorkerId
	job.ClientId = c.ClientId
	job.FuncName = c.Req.Handle
	job.Params   = c.Req.Params
	if IsMulParams(job.Params) {
		job.ParamsType = PARAMS_TYPE_MUL
	} else {
		job.ParamsType = PARAMS_TYPE_ONE
	}

	if ok := worker.Jobs.PushList(job); ok {
		worker.Lock()
		worker.JobNum++
		worker.Unlock()
	}

	go worker.doWork()

	return
}

func (c *SClient) RunClient() {
	dataType := c.Req.GetReqDataType()

	switch dataType {
		case PDT_C_DO_JOB:
		{
			go c.doJob()
		}
	}
}