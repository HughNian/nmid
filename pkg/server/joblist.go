package server

import (
	"container/list"
	"net/http"
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/model"
	"github.com/google/uuid"
)

//**joblist file use golang package container/list**//
//**this joblist file can replace job file's features if you want**//

type JobData struct {
	sync.Mutex

	Sw8    string
	Sw8Len uint32

	JobId       string //funcName + Params + time
	ClientId    string
	Client      *Connect
	HTTPClientR *http.Request
	WorkerId    string

	status uint32
	// weight int

	FuncName         string
	ParamsType       uint32
	ParamsHandleType uint32
	Params           []byte
	RetData          []byte
	BeginDuring      time.Time
}

type JobDataList struct {
	sync.Mutex

	JList *list.List
}

func NewJobData(Handle, Params string) (data *JobData) {
	data = new(JobData)

	data.JobId = uuid.Must(uuid.NewRandom()).String() //utils.GetJobId(Handle, Params)
	data.status = model.JOB_STATUS_INIT
	data.BeginDuring = time.Now()

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

func (jlist *JobDataList) GetJobNum() int {
	jlist.Lock()
	defer jlist.Unlock()

	return jlist.JList.Len()
}

func (jlist *JobDataList) PushJobData(data *JobData) bool {
	if data == nil {
		return false
	}

	jlist.Lock()
	jlist.JList.PushBack(data)
	jlist.Unlock()

	return true
}

func (jlist *JobDataList) PopJobData() (data *JobData) {
	jlist.Lock()
	item := jlist.JList.Back()
	if item == nil {
		jlist.Unlock()
		return nil
	}

	data = item.Value.(*JobData)
	jlist.JList.Remove(item)
	jlist.Unlock()

	return data
}

func (jlist *JobDataList) UnShiftJobData(data *JobData) bool {
	if data == nil {
		return false
	}

	jlist.Lock()
	jlist.JList.PushFront(data)
	jlist.Unlock()

	return true
}

func (jlist *JobDataList) ShiftJobData() (data *JobData) {
	jlist.Lock()
	item := jlist.JList.Front()
	if item == nil {
		jlist.Unlock()
		return nil
	}

	data = item.Value.(*JobData)
	jlist.JList.Remove(item)
	jlist.Unlock()

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
