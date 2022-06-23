package server

import (
	"fmt"
	"sync"

	"github.com/joshbohde/codel"
	"github.com/juju/ratelimit"
)

type SWorker struct {
	sync.Mutex

	WorkerId string
	Connect  *Connect

	JobNum   int
	Jobs     *JobDataList
	DingJobs *JobDataList
	DoneJobs *JobDataList

	Req *Request
	Res *Response

	NoJobNums int
	Sleep     bool

	CodelLimiter  *codel.Lock
	BucketLimiter *ratelimit.Bucket
}

func NewSWorker(conn *Connect) *SWorker {
	if conn == nil {
		return nil
	}

	return &SWorker{
		WorkerId:      conn.Id,
		Connect:       conn,
		JobNum:        0,
		Jobs:          NewJobDataList(),
		Req:           NewReq(),
		Res:           NewRes(),
		NoJobNums:     0,
		Sleep:         false,
		CodelLimiter:  NewCodelLimiter(),
		BucketLimiter: NewBucketLimiter(),
	}
}

func (w *SWorker) addFunction() {
	if w.Req.DataLen > 0 {
		functionName := w.Req.GetReqData()

		if len(functionName) != 0 {
			w.Connect.Ser.Funcs.AddFunc(w, string(functionName))
		}
	}

	w.Res.DataType = PDT_OK
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
	fmt.Println(`server do work`)
	fmt.Println(`server do work job id`, job.JobId)
	if w.JobNum > 0 {
		if job != nil && job.WorkerId == w.WorkerId && job.status == JOB_STATUS_INIT {
			job.status = JOB_STATUS_DOING

			functionName := job.FuncName
			params := job.Params
			paramsLen := uint32(len(params))
			if functionName != `` && paramsLen != 0 {
				w.Res.DataType = PDT_S_GET_DATA
				w.Res.Handle = functionName
				w.Res.HandleLen = uint32(len(functionName))
				if IsMulParams(params) {
					w.Res.ParamsType = PARAMS_TYPE_MUL
				} else {
					w.Res.ParamsType = PARAMS_TYPE_ONE
				}
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
		if job != nil && job.WorkerId == w.WorkerId && job.status == JOB_STATUS_DOING {
			//任务完成判断
			if w.Res.HandleLen == w.Req.HandleLen &&
				w.Res.Handle == w.Req.Handle &&
				w.Res.ParamsLen == w.Req.ParamsLen &&
				string(w.Res.Params) == string(w.Req.Params) {
				job.RetData = append(job.RetData, w.Req.Ret...)
				job.status = JOB_STATUS_DONE
			} else {
				return
			}

			clientId := job.ClientId
			functionName := job.FuncName
			params := job.Params
			paramsLen := len(params)
			if clientId != `` && functionName != `` && paramsLen != 0 {
				if client := w.Connect.Ser.Cpool.GetConnect(clientId); client != nil {
					fmt.Println(`server returndata job num`, w.JobNum)
					fmt.Println(`returndata client id`, clientId)
					fmt.Println(`server returndata getjob job id`, job.JobId)
					w.Res.DataType = PDT_S_RETURN_DATA
					w.Res.Ret = job.RetData
					w.Res.RetLen = w.Req.RetLen

					if client.ConnType == CONN_TYPE_CLIENT {
						client.Lock()
						resPack := w.Res.ResEncodePack()
						client.Write(resPack)
						client.Unlock()
					}
				}
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
	w.Res.DataType = PDT_RATELIMIT
	resPack := w.Res.ResEncodePack()
	w.Connect.Write(resPack)
}

//runworker 此处做熔断操作
func (w *SWorker) RunWorker() {
	//if !DoBucketLimiter(w.BucketLimiter) { //令牌桶限流
	//	w.doLimit()
	//} else {
	dataType := w.Req.GetReqDataType()

	switch dataType {
	//worker add function
	case PDT_W_ADD_FUNC:
		{
			w.addFunction()
		}
	//worker del function
	case PDT_W_DEL_FUNC:
		{
			w.delFunction()
		}
	case PDT_WAKEUP:
		{
			w.workerWakeup()
		}
	//worker grab job
	case PDT_W_GRAB_JOB:
		{
			//go w.doWork(``)
		}
	//worker return data
	case PDT_W_RETURN_DATA:
		{
			w.returnData()
		}
	}
	//}
}
