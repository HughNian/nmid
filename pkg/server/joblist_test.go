package server

import (
	"fmt"
	"strconv"
	"testing"
)

func TestAddJobData(t *testing.T) {
	data := NewJobData(`handler`, `params`)
	data.SetJobDataClient(`c1`)
	data.SetJobDataWorker(`w1`)

	jlist := NewJobDataList()
	jlist.PushJobData(data)
}

func BenchmarkAddJobData(b *testing.B) {
	data := NewJobData(`handler`, `params`)
	data.SetJobDataClient(`c1`)
	data.SetJobDataWorker(`w1`)

	jlist := NewJobDataList()
	jlist.PushJobData(data)
}

func TestGetFirstJobData(t *testing.T) {
	jlist := NewJobDataList()

	for i := 0; i < 10000; i++ {
		handler := `handler` + strconv.Itoa(i)
		params := `params` + strconv.Itoa(i)
		data := NewJobData(handler, params)
		data.SetJobDataClient(`c1`)
		data.SetJobDataWorker(`w1`)

		jlist.PushJobData(data)
	}

	getData := jlist.PopJobData()
	if getData != nil {
		fmt.Println(getData.FuncName)
	}
}
