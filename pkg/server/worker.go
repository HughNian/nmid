package server

import (
	"nmid-v2/pkg/conf"
	"sync"

	"github.com/joshbohde/codel"
	"github.com/juju/ratelimit"
)

type SWorker struct {
	sync.Mutex

	WorkerId string
	Connect  *Connect

	JobNum int
	Weight uint

	Jobs     *JobDataList
	DingJobs *JobDataList
	DoneJobs *JobDataList

	Req *Request
	Res *Response

	NoJobNums int
	Sleep     bool

	CodelLimiter  *codel.Lock
	BucketLimiter *ratelimit.Bucket

	HttpResTag chan struct{}
}

func NewSWorker(conn *Connect) *SWorker {
	if conn == nil {
		return nil
	}

	return &SWorker{
		WorkerId:      conn.Id,
		Connect:       conn,
		Jobs:          NewJobDataList(),
		Req:           NewReq(),
		Res:           NewRes(),
		NoJobNums:     0,
		Sleep:         false,
		CodelLimiter:  NewCodelLimiter(),
		BucketLimiter: NewBucketLimiter(),
		HttpResTag:    make(chan struct{}),
	}
}

func (w *SWorker) addFunction() {
	if w.Req.DataLen > 0 {
		functionName := w.Req.GetReqData()

		if len(functionName) != 0 {
			w.Connect.Ser.Funcs.AddFunc(w, string(functionName))
		}
	}

	w.Res.DataType = conf.PDT_OK
	resPack := w.Res.ResEncodePack()
	w.Connect.Write(resPack)
}

func (w *SWorker) delFunction() {
	if w.Req.DataLen > 0 {
		functionName := w.Req.GetReqData()

		if len(functionName) != 0 {
			w.Connect.Ser.Funcs.DelWorkerFunc(w.WorkerId, string(functionName))
		}
	}
}

func (w *SWorker) delWorkerJob(status uint32) {
	if w.JobNum > 0 {
		w.Jobs.DelJobDataStats(status)
		w.Lock()
		w.JobNum--
		w.Unlock()
	}
}

func (w *SWorker) delWorkerJobV2(jobId string) {
	if w.JobNum > 0 {
		w.Jobs.DelJobData(jobId)
		w.Lock()
		w.JobNum--
		w.Unlock()
	}
}

func (w *SWorker) doWork(job *JobData) {
	if w.JobNum > 0 {
		if job != nil && job.WorkerId == w.WorkerId && job.status == conf.JOB_STATUS_INIT {
			job.status = conf.JOB_STATUS_DOING

			functionName := job.FuncName
			params := job.Params
			paramsLen := uint32(len(params))
			if functionName != `` && paramsLen != 0 {
				w.Res.DataType = conf.PDT_S_GET_DATA
				w.Res.Handle = functionName
				w.Res.HandleLen = uint32(len(functionName))
				w.Res.ParamsType = job.ParamsType
				w.Res.ParamsLen = paramsLen
				w.Res.Params = params //append(w.Res.Params, params...)
				w.Res.JobId = job.JobId
				w.Res.JobIdLen = uint32(len(job.JobId))

				resPack := w.Res.ResEncodePack()
				w.Connect.Lock()
				w.Connect.Write(resPack)
				w.Connect.Unlock()
			}
		}
	}
}

func (w *SWorker) returnData() {
	if w.JobNum > 0 {
		//解包获取数据内容
		w.Req.ReqDecodePack()
		job := w.Jobs.GetJobData(w.Req.JobId)
		if job != nil && job.WorkerId == w.WorkerId && job.status == conf.JOB_STATUS_DOING {
			//任务完成判断
			if w.Res.HandleLen == w.Req.HandleLen &&
				w.Res.Handle == w.Req.Handle &&
				w.Res.ParamsLen == w.Req.ParamsLen &&
				string(w.Res.Params) == string(w.Req.Params) {
				job.RetData = append(job.RetData, w.Req.Ret...)
				job.status = conf.JOB_STATUS_DONE
			} else {
				return
			}

			clientId := job.ClientId
			functionName := job.FuncName
			params := job.Params
			paramsLen := len(params)
			if clientId != `` && functionName != `` && paramsLen != 0 {
				//tcp client response
				if client := w.Connect.Ser.Cpool.GetConnect(clientId); client != nil {
					w.Res.DataType = conf.PDT_S_RETURN_DATA
					w.Res.Ret = job.RetData
					w.Res.RetLen = w.Req.RetLen

					if client.ConnType == conf.CONN_TYPE_CLIENT {
						client.Lock()
						resPack := w.Res.ResEncodePack()
						client.Write(resPack)
						client.Unlock()
					}
				}
			} else if job.HTTPClientR != nil && functionName != `` && paramsLen != 0 {
				//http client response data
				w.Res.DataType = conf.PDT_S_RETURN_DATA
				w.Res.Ret = job.RetData
				w.Res.RetLen = w.Req.RetLen

				w.HttpResTag <- struct{}{}
			}
		}

		w.delWorkerJobV2(w.Req.JobId)
	}
}

func (w *SWorker) workerWakeup() {
	w.Sleep = false
	w.NoJobNums = 0
}

func (w *SWorker) doLimit() {
	w.Res.DataType = conf.PDT_RATELIMIT
	resPack := w.Res.ResEncodePack()
	w.Connect.Write(resPack)
}

func (w *SWorker) RunWorker() {
	//if !DoBucketLimiter(w.BucketLimiter) { //令牌桶限流
	//	w.doLimit()
	//} else {
	dataType := w.Req.GetReqDataType()

	switch dataType {
	//worker add function
	case conf.PDT_W_ADD_FUNC:
		{
			w.addFunction()
		}
	//worker del function
	case conf.PDT_W_DEL_FUNC:
		{
			w.delFunction()
		}
	case conf.PDT_WAKEUP:
		{
			w.workerWakeup()
		}
	//worker grab job
	case conf.PDT_W_GRAB_JOB:
		{
			//go w.doWork(``)
		}
	//worker return data
	case conf.PDT_W_RETURN_DATA:
		{
			w.returnData()
		}
	}
	//}
}
