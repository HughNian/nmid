package server

import (
	"sync"
)

type Job struct {
	sync.Mutex

	JobId      string //funcName + Params + time
	ClientId   string
	WorkerId   string

	status     uint32
	weight     int

	FuncName   string
	ParamsType uint32
	Params     []byte
	RetData    []byte

	Prev *Job
	Next *Job
}

type JobList struct {
	sync.Mutex

	Head *Job
	Size uint32
}

func NewJob(Handle, Params string) (j *Job) {
	j = new (Job)

	j.JobId = GetJobId(Handle, Params)
	j.ClientId = ``
	j.WorkerId = ``
	j.status   = JOB_STATUS_INIT
	j.FuncName = ``
	j.Params   = make([]byte, 0)
	j.RetData  = make([]byte, 0)
	j.Prev     = nil
	j.Next     = nil

	return j
}

func NewJobList() (jl *JobList) {
	jl = new (JobList)

	jl.Head = nil
	jl.Size = 0

	return jl
}

func (j *Job) SetJobClient(id string) {
	j.Lock()
	defer j.Unlock()

	j.ClientId = id
}

func (j *Job) SetJobWorker(id string) {
	j.Lock()
	defer j.Unlock()

	j.WorkerId = id
}

func (jl *JobList) PushList(job *Job) bool {
	jl.Lock()
	defer jl.Unlock()

	if job == nil {
		return false
	}

	tmpHead := jl.Head

	if jl.Size == 0 || tmpHead == nil {
		jl.Head = job
		jl.Head.Prev = nil
		jl.Head.Next = nil
	} else if tmpHead != nil && tmpHead.Next == nil {
		job.Next = nil
		job.Prev = tmpHead
		tmpHead.Next = job
	} else if tmpHead.Next != nil {
		//find the tail
		tmpNext := tmpHead.Next
		for tmpNext.Next != nil {
			tmpNext = tmpNext.Next
		}

		job.Next = nil
		job.Prev = tmpNext
		tmpNext.Next = job
	}

	jl.Size++

	return true
}

func (jl *JobList) PopList() (job *Job) {
	jl.Lock()
	defer jl.Unlock()

	if jl.Size == 0 || jl.Head == nil {
		return nil
	}

	job = jl.Head

	if jl.Size > 1 {
		if job.Prev != nil || job.Next == nil {
			return nil
		}
		nextJob := job.Next
		nextJob.Prev = nil
		jl.Head = nextJob
	}

	job.Prev, job.Next = nil, nil
	jl.Size--

	return
}

func (jl *JobList) DeListJob(jobId string) bool {
	jl.Lock()
	defer jl.Unlock()

	if jl.Size == 0 || jl.Head == nil {
		return false
	}

	var i uint32
	i = 0
	job := jl.Head
	for {
		if jl.Size < i {
			break
		}
		if job.JobId == jobId {
			prevJob := job.Prev
			nextJob := job.Next
			prevJob.Next = nextJob
			nextJob.Prev = prevJob
			break
		}
		job = job.Next
		i++
	}

	return true
}

func (jl *JobList) DelListStatsJob(status uint32) (delNum int) {
	jl.Lock()
	defer jl.Unlock()

	if jl.Size == 0 || jl.Head == nil {
		return 0
	}

	var i uint32
	i = 0
	delNum = 0
	job := jl.Head
	for ; i < jl.Size; i ++ {
		if job.status == status {
			prevJob := job.Prev
			nextJob := job.Next
			if jl.Size > 1 {
				prevJob.Next = nextJob
				nextJob.Prev = prevJob
			}

			delNum++
		}
		job = job.Next
	}
	jl.Size -= uint32(delNum)

	return delNum
}

func (jl *JobList) GetListJob(jobId string) (job *Job) {
	jl.Lock()
	defer jl.Unlock()

	if jl.Size == 0 || jl.Head == nil {
		return nil
	}

	var i uint32
	i = 0
	isGet := 0
	job = jl.Head
	for {
		if jl.Size < i {
			break
		}
		if job.JobId == jobId {
			isGet = 1
			break
		}
		job = job.Next
		i++
	}

	if isGet == 1 {
		return job
	} else {
		return nil
	}
}