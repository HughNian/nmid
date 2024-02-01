package server

import (
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/limiter"
	"github.com/HughNian/nmid/pkg/model"

	"github.com/joshbohde/codel"
	"github.com/juju/ratelimit"
)

type SWorker struct {
	sync.Mutex

	WorkerId string
	Connect  *Connect

	Weight uint

	Jobs     *JobDataList
	DingJobs *JobDataList
	DoneJobs *JobDataList

	JobsMap sync.Map

	Req *Request
	Res *Response

	Sleep bool

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
		Sleep:         false,
		CodelLimiter:  limiter.NewCodelLimiter(),
		BucketLimiter: limiter.NewBucketLimiter(),
		HttpResTag:    make(chan struct{}),
	}
}

func (w *SWorker) addFunction() {
	if w.Req.DataLen > 0 {
		functionName := w.Req.GetReqData()

		if len(functionName) != 0 {
			w.Connect.Ser.Funcs.AddFunc(w, string(functionName))
		}

		//do prometheus worker func count
		WorkerFuncCount.Add(1, w.WorkerId, string(functionName))

		//do prometheus func count
		FuncCount.Add(1, string(functionName))
	}

	w.Res.DataType = model.PDT_OK
	resPack := w.Res.ResEncodePack()
	w.Connect.Write(resPack)
}

func (w *SWorker) delFunction() {
	if w.Req.DataLen > 0 {
		functionName := w.Req.GetReqData()

		if len(functionName) != 0 {
			w.Connect.Ser.Funcs.DelWorkerFunc(w.WorkerId, string(functionName))

			//do prometheus worker func count
			WorkerFuncCount.Add(-1, w.WorkerId, string(functionName))

			//do prometheus func count
			FuncCount.Add(-1, string(functionName))
		}
	}
}

func (w *SWorker) doWork(job *JobData) {
	if job != nil && job.WorkerId == w.WorkerId && job.status == model.JOB_STATUS_INIT {
		job.status = model.JOB_STATUS_DOING

		functionName := job.FuncName
		params := job.Params
		paramsLen := uint32(len(params))
		if functionName != `` && paramsLen != 0 {
			w.Res.DataType = model.PDT_S_GET_DATA
			w.Res.Handle = functionName
			w.Res.HandleLen = uint32(len(functionName))
			w.Res.ParamsType = job.ParamsType
			w.Res.ParamsHandleType = job.ParamsHandleType
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

func (w *SWorker) returnData() {
	job := w.Jobs.PopJobData()
	// var job *JobData
	// ret, ok := w.JobsMap.Load(w.Req.JobId)
	// if ok {
	// 	job = ret.(*JobData)
	// 	w.JobsMap.Delete(w.Req.JobId)
	// }

	if job != nil && job.WorkerId == w.WorkerId && job.status == model.JOB_STATUS_DOING {
		//解包获取数据内容
		w.Req.ReqDecodePack()

		//任务完成判断
		if w.Res.HandleLen == w.Req.HandleLen &&
			w.Res.Handle == w.Req.Handle &&
			w.Res.ParamsLen == w.Req.ParamsLen &&
			string(w.Res.Params) == string(w.Req.Params) {
			job.RetData = append(job.RetData, w.Req.Ret...)
			job.status = model.JOB_STATUS_DONE
		} else {
			return
		}

		clientId := job.ClientId
		functionName := job.FuncName
		params := job.Params
		paramsLen := len(params)
		if clientId != `` && functionName != `` && paramsLen != 0 {
			//tcp client response
			if client := job.Client; client != nil {
				w.Res.DataType = model.PDT_S_RETURN_DATA
				w.Res.Ret = job.RetData
				w.Res.RetLen = w.Req.RetLen

				if client.ConnType == model.CONN_TYPE_CLIENT {
					client.Lock()
					resPack := w.Res.ResEncodePack()
					werr := client.Write(resPack)
					client.Unlock()

					//do prometheus worker write success/fail num
					if werr == nil {
						//success num
						WorkerFuncSuccessCount.Inc(w.WorkerId, functionName)
						FuncSuccessCount.Inc(functionName)
					} else {
						//fail num
						WorkerFuncFailCount.Inc(w.WorkerId, functionName)
						FuncFailCount.Inc(functionName)
					}
				}

				//do prometheus request during
				ftime := float64(time.Since(job.BeginDuring).Milliseconds())
				requestDuring.Add(ftime, w.WorkerId, functionName)

				//do prometheus request duration
				requestDuration.Observe(time.Since(job.BeginDuring).Milliseconds(), w.WorkerId, functionName)
			}
		} else if job.HTTPClientR != nil && functionName != `` && paramsLen != 0 {
			//http client response data
			w.Res.DataType = model.PDT_S_RETURN_DATA
			w.Res.Ret = job.RetData
			w.Res.RetLen = w.Req.RetLen

			w.HttpResTag <- struct{}{}
		}
	}
}

func (w *SWorker) workerWakeup() {
	w.Sleep = false
}

func (w *SWorker) doLimit() {
	w.Res.DataType = model.PDT_RATELIMIT
	resPack := w.Res.ResEncodePack()
	w.Connect.Write(resPack)
}

func (w *SWorker) heartBeatPong() {
	// logger.Infof("server worker heartbeat pong")
	w.Res.DataType = model.PDT_S_HEARTBEAT_PONG
	resPack := w.Res.ResEncodePack()
	w.Connect.Write(resPack)
}

func (w *SWorker) RunWorker() {
	dataType := w.Req.GetReqDataType()

	switch dataType {
	//worker add function
	case model.PDT_W_ADD_FUNC:
		{
			w.addFunction()
		}
	//worker del function
	case model.PDT_W_DEL_FUNC:
		{
			w.delFunction()
		}
	case model.PDT_WAKEUP:
		{
			w.workerWakeup()
		}
	//worker grab job
	case model.PDT_W_GRAB_JOB:
		{
			//todo get job list one
		}
	//worker return data
	case model.PDT_W_RETURN_DATA:
		{
			w.returnData()
		}
	//heartbeat
	case model.PDT_W_HEARTBEAT_PING:
		{
			w.heartBeatPong()
		}
	}
}

func (w *SWorker) CloseSelfWorker() {
	w.Connect.Ser.Funcs.CleanWorkerFunc(w.WorkerId)
	if w.Connect.Conn != nil {
		w.Connect.Conn.Close()
	}
}
