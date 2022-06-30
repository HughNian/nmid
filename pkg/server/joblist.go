package server

import (
	"container/list"
	"net/http"
	"sync"
)

//**joblist file use golang package container/list**//
//**this joblist file can replace job file's features if you want**//

type JobData struct {
	sync.Mutex

	JobId       string //funcName + Params + time
	ClientId    string
	HTTPClientR *http.Request
	HTTPClientW http.ResponseWriter
	WorkerId    string

	status uint32
	// weight int

	FuncName   string
	ParamsType uint32
	Params     []byte
	RetData    []byte
}

type JobDataList struct {
	sync.Mutex

	JList *list.List
}

func NewJobData(Handle, Params string) (data *JobData) {
	data = new(JobData)

	data.JobId = GetJobId(Handle, Params)
	data.status = JOB_STATUS_INIT

	return data
}

func NewJobDataList() (jlist *JobDataList) {
	jlist = &JobDataList{
		JList: list.New(),
	}

	return jlist
}

func (data *JobData) SetJobDataClient(id string) {
	data.Lock()
	defer data.Unlock()

	data.ClientId = id
}

func (data *JobData) SetJobDataWorker(id string) {
	data.Lock()
	defer data.Unlock()

	data.WorkerId = id
}

func (jlist *JobDataList) PushJobData(data *JobData) bool {
	if data == nil {
		return false
	}

	jlist.Lock()
	defer jlist.Unlock()
	jlist.JList.PushBack(data)

	return true
}

func (jlist *JobDataList) PopJobData() (data *JobData) {
	jlist.Lock()
	defer jlist.Unlock()

	item := jlist.JList.Front()
	if item == nil {
		return nil
	}

	data = item.Value.(*JobData)

	return data
}

func (jlist *JobDataList) DelJobData(jobId string) bool {
	jlist.Lock()
	defer jlist.Unlock()

	for d := jlist.JList.Front(); d != nil; d = d.Next() {
		data := d.Value.(*JobData)
		if data.JobId == jobId {
			jlist.JList.Remove(d)
			return true
		}
	}

	return false
}

func (jlist *JobDataList) DelJobDataStats(status uint32) (delNum int) {
	jlist.Lock()
	defer jlist.Unlock()

	for d := jlist.JList.Front(); d != nil; d = d.Next() {
		data := d.Value.(*JobData)
		if data.status == status {
			jlist.JList.Remove(d)
			delNum++
		}
	}

	return delNum
}

func (jlist *JobDataList) GetJobData(jobId string) (data *JobData) {
	for d := jlist.JList.Front(); d != nil; d = d.Next() {
		data = d.Value.(*JobData)
		if data.JobId == jobId {
			return data
		}
	}

	return nil
}
