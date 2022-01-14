package server

import (
	"fmt"
	"strconv"
	"testing"
)

func TestAddJob(t *testing.T) {
	jobs := NewJobList()
	job := NewJob(`testhandler`, `params`)
	job.WorkerId = `111`
	job.ClientId = `222`
	job.FuncName = `testhandler`
	job.Params = []byte(`params`)

	jobs.PushList(job)
}

func BenchmarkAddJob(b *testing.B) {
	jobs := NewJobList()
	job := NewJob(`testhandler`, `params`)
	job.WorkerId = `111`
	job.ClientId = `222`
	job.FuncName = `testhandler`
	job.Params = []byte(`params`)

	jobs.PushList(job)
}

func BenchmarkPop(b *testing.B) {
	jobs := NewJobList()
	for i := 0; i < 100; i++ {
		job := NewJob(`testhandler`, `params`)
		job.WorkerId = `111` + strconv.Itoa(i)
		job.ClientId = `222` + strconv.Itoa(i)
		job.FuncName = `testhandler` + strconv.Itoa(i)
		job.Params = []byte(`params`)

		jobs.PushList(job)
		jobs.PopList()
	}
}

func BenchmarkDelStatasJob(b *testing.B) {
	jobs := NewJobList()

	job := NewJob(`testhandler`, `params`)
	job.JobId = "1"
	job.WorkerId = `111` + strconv.Itoa(1)
	job.ClientId = `222` + strconv.Itoa(1)
	job.FuncName = `testhandler` + strconv.Itoa(1)
	job.Params = []byte(`params`)
	job.status = 2
	jobs.PushList(job)

	for i := 1; i < 100000; i++ {
		job := NewJob(`testhandler`, `params`)
		job.JobId = strconv.Itoa(i)
		job.WorkerId = `111` + strconv.Itoa(i)
		job.ClientId = `222` + strconv.Itoa(i)
		job.FuncName = `testhandler` + strconv.Itoa(i)
		job.Params = []byte(`params`)
		job.status = 1

		jobs.PushList(job)
		jobs.DelListStatsJob(1)
	}

	job2 := jobs.GetListJob("1")
	if job2 != nil {
		fmt.Println(job2.JobId)
	} else {
		fmt.Println("no data")
	}
}
