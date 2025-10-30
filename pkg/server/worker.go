package server

import (
	"fmt"
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/limiter"
	"github.com/HughNian/nmid/pkg/model"

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

	Req *Request
	Res *Response

	Sleep bool

	CodelLimiter  *codel.Lock
	BucketLimiter *ratelimit.Bucket

	HttpResTag chan struct{}
}

type ResultJob struct {
	JobId    string
	FuncName string
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
		Req:           NewReq(),
		Res:           NewRes(),
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
		w.returnData(result.FuncName)
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

func (w *SWorker) addFunction() {
	if w.Req.DataLen > 0 {
		functionName := w.Req.GetReqData()

		if len(functionName) != 0 {
			w.Connect.Ser.Funcs.AddFunc(w, string(functionName))
		}

		//do prometheus worker func count
		createTime := time.Now().Format("2006-01-02 15:04:05.000000")
		WorkerFuncCount.Add(1, w.WorkerId, w.WorkerName, w.Connect.Ip, string(functionName), createTime)

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
			go func() {
				w.Connect.Lock()
				w.Connect.Write(resPack)
				w.Connect.Unlock()
			}()
		}
	}
}

func (w *SWorker) returnData(handle string) {
	job := w.Jobs[handle].PopJobData()
	// job := w.PopJobFromChannel(handle)

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
			fmt.Println("res handle len", w.Res.HandleLen)
			fmt.Println("req handle len", w.Req.HandleLen)
			fmt.Println("res handle", w.Res.Handle)
			fmt.Println("req handle", w.Req.Handle)
			fmt.Println("res params", string(w.Res.Params))
			fmt.Println("req params", string(w.Req.Params))
			return
		}

		clientId := job.ClientId
		functionName := job.FuncName
		params := job.Params
		paramsLen := len(params)
		if clientId != `` && functionName != `` && paramsLen != 0 {
			//tcp client response
			if client := job.Client; client != nil {
				res := *w.Res
				res.DataType = model.PDT_S_RETURN_DATA
				res.Ret = job.RetData
				res.RetLen = w.Req.RetLen

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

func (w *SWorker) setWorkerName() {
	if w.Req.DataLen > 0 {
		workerName := w.Req.GetReqData()

		if len(workerName) != 0 {
			w.WorkerName = string(workerName)
		}
	}

	w.Res.DataType = model.PDT_OK
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
			select {
			case w.Results <- &ResultJob{
				FuncName: w.Res.Handle,
				JobId:    w.Res.JobId,
			}:
			default:
				w.returnData(w.Res.Handle)
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
			w.setWorkerName()
		}
	}
}

func (w *SWorker) CloseSelfWorker() {
	w.Connect.Ser.Funcs.CleanWorkerFunc(w.WorkerId)
	if w.Connect.Conn != nil {
		w.Connect.Conn.Close()
	}
}
