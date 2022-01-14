package server

import (
	"sync"
)

type SWorker struct {
	sync.Mutex

	WorkerId string
	Connect  *Connect

	JobNum   int
	Jobs     *JobList
	DingJobs *JobList
	DoneJobs *JobList

	Req *Request
	Res *Response

	NoJobNums int
	Sleep     bool
}

func NewSWorker(conn *Connect) *SWorker {
	if conn == nil {
		return nil
	}

	return &SWorker{
		WorkerId:  conn.Id,
		Connect:   conn,
		JobNum:    0,
		Jobs:      NewJobList(),
		DingJobs:  NewJobList(),
		DoneJobs:  NewJobList(),
		Req:       NewReq(),
		Res:       NewRes(),
		NoJobNums: 0,
		Sleep:     false,
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

func (w *SWorker) delWorkerJob() {
	if w.JobNum > 0 {
		doneNum := w.DoneJobs.DelListStatsJob(JOB_STATUS_DONE)
		w.Lock()
		w.JobNum -= doneNum
		w.Unlock()
	}
}

func (w *SWorker) doWork() {
	if w.JobNum > 0 {
		job := w.Jobs.PopList()
		if job != nil && job.WorkerId == w.WorkerId && job.status == JOB_STATUS_INIT {
			job.status = JOB_STATUS_DOING
			w.DingJobs.PushList(job)

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

				resPack := w.Res.ResEncodePack()
				w.Connect.Lock()
				w.Connect.Write(resPack)
				w.Connect.Unlock()
			}
		}
	} /*else {
		w.Res.DataType = PDT_NO_JOB
		w.Lock()
		w.NoJobNums++
		w.Unlock()
		fmt.Println("######NoJobNums-", w.NoJobNums)

		if w.NoJobNums >= MAX_NOJOB_NUM {
			w.Sleep = true
			w.Res.DataType = PDT_TOSLEEP
		}
		fmt.Println("datatype:", w.Res.DataType)
		resPack := w.Res.ResEncodePack()
		w.Connect.Write(resPack)
	}*/
}

func (w *SWorker) returnData() {
	if w.JobNum > 0 {
		job := w.DingJobs.PopList()
		if job != nil && job.WorkerId == w.WorkerId && job.status == JOB_STATUS_DOING {
			//解包获取数据内容
			w.Req.ReqDecodePack()
			//任务完成判断
			if w.Res.HandleLen == w.Req.HandleLen &&
				w.Res.Handle == w.Req.Handle &&
				w.Res.ParamsLen == w.Req.ParamsLen &&
				string(w.Res.Params) == string(w.Req.Params) {
				job.RetData = append(job.RetData, w.Req.Ret...)
				job.status = JOB_STATUS_DONE
				w.DoneJobs.PushList(job)
			} else {
				return
			}

			clientId := job.ClientId
			functionName := job.FuncName
			params := job.Params
			paramsLen := len(params)
			if clientId != `` && functionName != `` && paramsLen != 0 {
				if client := w.Connect.Ser.Cpool.GetConnect(clientId); client != nil {
					w.Res.DataType = PDT_S_RETURN_DATA
					w.Res.Ret = job.RetData
					w.Res.RetLen = w.Req.RetLen

					resPack := w.Res.ResEncodePack()
					client.Lock()
					client.Write(resPack)
					client.Unlock()
				}
			}
		}

		w.delWorkerJob()
	}
}

func (w *SWorker) workerWakeup() {
	w.Sleep = false
	w.NoJobNums = 0
}

func (w *SWorker) RunWorker() {
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
			go w.doWork()
		}
	//worker return data
	case PDT_W_RETURN_DATA:
		{
			go w.returnData()
		}
	}
}
