package server

import (
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/limiter"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/go-kratos/kratos/pkg/log"

	"github.com/joshbohde/codel"
	"github.com/juju/ratelimit"
)

type SWorker struct {
	sync.Mutex

	WorkerId   string
	WorkerName string
	Connect    *Connect

	Weight uint

	Jobs map[string]*JobDataList

	JobChannelsMutex sync.RWMutex
	JobChannels      map[string]chan *JobData

	Results chan *ResultJob

	Sleep bool

	CodelLimiter  *codel.Lock
	BucketLimiter *ratelimit.Bucket

	HttpResTag chan struct{}
}

type ResultJob struct {
	JobId    string
	FuncName string
	Request  *Request
}

func NewSWorker(conn *Connect) *SWorker {
	if conn == nil {
		return nil
	}

	sworker := &SWorker{
		WorkerId:      conn.Id,
		Connect:       conn,
		Jobs:          make(map[string]*JobDataList),
		JobChannels:   make(map[string]chan *JobData),
		Results:       make(chan *ResultJob, 1024),
		Sleep:         false,
		CodelLimiter:  limiter.NewCodelLimiter(),
		BucketLimiter: limiter.NewBucketLimiter(),
		HttpResTag:    make(chan struct{}),
	}

	go sworker.processResults()

	return sworker
}

func (w *SWorker) processResults() {
	for result := range w.Results {
		w.returnData(result)
	}
}

func (w *SWorker) GetJobChannel(funcName string) chan *JobData {
	if ch, exists := w.JobChannels[funcName]; exists {
		return ch
	}

	ch := make(chan *JobData, 1024) // 缓冲 channel
	w.JobChannels[funcName] = ch
	return ch
}

func (w *SWorker) PushJobToChannel(job *JobData) bool {
	if job == nil || job.FuncName == "" {
		return false
	}

	ch := w.GetJobChannel(job.FuncName)

	select {
	case ch <- job:
		return true
	default:
		// channel 满了，可以选择丢弃或阻塞
		return false
	}
}

func (w *SWorker) PopJobFromChannel(handle string) *JobData {
	ch := w.GetJobChannel(handle)

	select {
	case job := <-ch:
		return job
	case <-time.After(50 * time.Millisecond):
		return nil
	}
}

func (w *SWorker) GetOrCreateJobList(funcName string) *JobDataList {
	if jobList, exists := w.Jobs[funcName]; exists {
		return jobList
	}

	jobList := NewJobDataList()
	w.Jobs[funcName] = jobList
	return jobList
}

func (w *SWorker) PushJobToList(job *JobData) bool {
	if job == nil || job.FuncName == "" {
		return false
	}

	jobList := w.GetOrCreateJobList(job.FuncName)
	return jobList.PushJobData(job)
}

func (w *SWorker) addFunction(req *Request) {
	if req.DataLen > 0 {
		functionName := req.GetReqData()

		if len(functionName) != 0 {
			w.Connect.Ser.Funcs.AddFunc(w, string(functionName))
		}

		//do prometheus worker func count
		createTime := time.Now().Format("2006-01-02 15:04:05.000000")
		WorkerFuncCount.Add(1, w.WorkerId, w.WorkerName, w.Connect.Ip, string(functionName), createTime)

		//do prometheus func count
		FuncCount.Add(1, string(functionName))
	}

	res := NewRes()
	res.DataType = model.PDT_OK
	resPack := res.ResEncodePack()
	w.Connect.Write(resPack)
}

func (w *SWorker) delFunction(req *Request) {
	if req.DataLen > 0 {
		functionName := req.GetReqData()

		if len(functionName) != 0 {
			w.Connect.Ser.Funcs.DelWorkerFunc(w.WorkerId, string(functionName))

			//do prometheus worker func count
			createTime := time.Now().Format("2006-01-02 15:04:05.000000")
			WorkerFuncCount.Add(-1, w.WorkerId, w.WorkerName, w.Connect.Ip, string(functionName), createTime)

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
			res := NewRes()
			res.DataType = model.PDT_S_GET_DATA
			res.Handle = functionName
			res.HandleLen = uint32(len(functionName))
			res.ParamsType = job.ParamsType
			res.ParamsHandleType = job.ParamsHandleType
			res.ParamsLen = paramsLen
			res.Params = params //append(w.Res.Params, params...)
			res.JobId = job.JobId
			res.JobIdLen = uint32(len(job.JobId))

			resPack := res.ResEncodePack()
			go func() {
				w.Connect.Lock()
				w.Connect.Write(resPack)
				w.Connect.Unlock()
			}()

			job.Response = res
		}
	}
}

func (w *SWorker) returnData(jobRet *ResultJob) {
	job := w.Jobs[jobRet.FuncName].PopJobData()
	// job := w.PopJobFromChannel(handle)
	req := jobRet.Request

	if job != nil && job.WorkerId == w.WorkerId && job.status == model.JOB_STATUS_DOING {
		//任务完成判断
		if job.Response != nil &&
			job.Response.HandleLen == req.HandleLen &&
			job.Response.Handle == req.Handle &&
			job.Response.ParamsLen == req.ParamsLen &&
			string(job.Response.Params) == string(req.Params) {
			job.RetData = append(job.RetData, req.Ret...)
			job.status = model.JOB_STATUS_DONE
		} else {
			log.Info("worker return data error")
			return
		}

		clientId := job.ClientId
		functionName := job.FuncName
		params := job.Params
		paramsLen := len(params)
		if clientId != `` && functionName != `` && paramsLen != 0 {
			//tcp client response
			if client := job.Client; client != nil {
				res := *job.Response
				res.DataType = model.PDT_S_RETURN_DATA
				res.Ret = job.RetData
				res.RetLen = req.RetLen

				if client.ConnType == model.CONN_TYPE_CLIENT {
					client.Lock()
					resPack := res.ResEncodePack()
					werr := client.Write(resPack)
					client.Unlock()

					//do prometheus worker write success/fail num
					if werr == nil {
						//success num
						WorkerFuncSuccessCount.Inc(w.WorkerName, functionName)
						FuncSuccessCount.Inc(functionName)
					} else {
						//fail num
						WorkerFuncFailCount.Inc(w.WorkerName, functionName)
						FuncFailCount.Inc(functionName)
					}
				}

				//do prometheus request during
				ftime := float64(time.Since(job.BeginDuring).Milliseconds())
				requestDuring.Add(ftime, w.WorkerName, functionName)

				//do prometheus request duration
				requestDuration.Observe(time.Since(job.BeginDuring).Milliseconds(), w.WorkerName, functionName)
			}
		} else if job.HTTPClientR != nil && functionName != `` && paramsLen != 0 {
			//http client response data
			job.Response.DataType = model.PDT_S_RETURN_DATA
			job.Response.Ret = job.RetData
			job.Response.RetLen = req.RetLen

			w.HttpResTag <- struct{}{}
		}
	}
}

func (w *SWorker) workerWakeup() {
	w.Sleep = false
}

func (w *SWorker) doLimit() {
	res := NewRes()
	res.DataType = model.PDT_RATELIMIT
	resPack := res.ResEncodePack()
	w.Connect.Write(resPack)
}

func (w *SWorker) heartBeatPong() {
	res := NewRes()
	// logger.Infof("server worker heartbeat pong")
	res.DataType = model.PDT_S_HEARTBEAT_PONG
	resPack := res.ResEncodePack()
	w.Connect.Write(resPack)
}

func (w *SWorker) setWorkerName(req *Request) {
	if req.DataLen > 0 {
		workerName := req.GetReqData()

		if len(workerName) != 0 {
			w.WorkerName = string(workerName)
		}
	}

	res := NewRes()
	res.DataType = model.PDT_OK
	resPack := res.ResEncodePack()
	w.Connect.Write(resPack)
}

func (w *SWorker) RunWorker(data []byte) {
	req := NewReq()
	req.DataType = w.Connect.DataType
	req.DataLen = w.Connect.DataLen
	req.Data = data

	//解包获取数据内容
	req.ReqDecodePack()

	switch req.DataType {
	//worker add function
	case model.PDT_W_ADD_FUNC:
		{
			w.addFunction(req)
		}
	//worker del function
	case model.PDT_W_DEL_FUNC:
		{
			w.delFunction(req)
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
			select {
			case w.Results <- &ResultJob{
				FuncName: req.Handle,
				JobId:    req.JobId,
				Request:  req,
			}:
			default:
				w.returnData(&ResultJob{
					FuncName: req.Handle,
					JobId:    req.JobId,
					Request:  req,
				})
			}
		}
	//heartbeat
	case model.PDT_W_HEARTBEAT_PING:
		{
			w.heartBeatPong()
		}
	//set worker name
	case model.PDT_W_SET_NAME:
		{
			w.setWorkerName(req)
		}
	}
}

func (w *SWorker) CloseSelfWorker() {
	w.Connect.Ser.Funcs.CleanWorkerFunc(w.WorkerId)
	if w.Connect.Conn != nil {
		w.Connect.Conn.Close()
	}
}
