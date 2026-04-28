package server

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/HughNian/nmid/pkg/model"
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

func BenchmarkConnectReadFrame(b *testing.B) {
	payload := make([]byte, 1024)
	frame := make([]byte, model.MIN_DATA_SIZE+len(payload))
	binary.BigEndian.PutUint32(frame[:4], model.CONN_TYPE_CLIENT)
	binary.BigEndian.PutUint32(frame[4:8], model.PDT_C_DO_JOB)
	binary.BigEndian.PutUint32(frame[8:model.MIN_DATA_SIZE], uint32(len(payload)))
	copy(frame[model.MIN_DATA_SIZE:], payload)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	c := &Connect{
		Conn:   serverConn,
		reader: bufio.NewReaderSize(serverConn, 32*1024),
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < b.N; i++ {
			_, _ = clientConn.Write(frame)
		}
		close(done)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err := c.ReadFrame()
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	<-done
}

func BenchmarkConnectWriteQueue(b *testing.B) {
	payload := make([]byte, 1024)
	frame := make([]byte, model.MIN_DATA_SIZE+len(payload))
	binary.BigEndian.PutUint32(frame[:4], model.CONN_TYPE_SERVER)
	binary.BigEndian.PutUint32(frame[4:8], model.PDT_S_RETURN_DATA)
	binary.BigEndian.PutUint32(frame[8:model.MIN_DATA_SIZE], uint32(len(payload)))
	copy(frame[model.MIN_DATA_SIZE:], payload)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	c := &Connect{
		Conn:    serverConn,
		writeCh: make(chan []byte, 4096),
		closeCh: make(chan struct{}),
	}
	go c.writeLoop()

	done := make(chan struct{})
	go func() {
		buf := make([]byte, 32*1024)
		for {
			_, err := clientConn.Read(buf)
			if err != nil {
				close(done)
				return
			}
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := c.Write(frame); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	c.closeSignal()
	_ = serverConn.Close()
	<-done
}
