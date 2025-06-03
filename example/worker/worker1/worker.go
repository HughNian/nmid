// nmid worker
//
// author: niansong(hugh.nian@163.com)
package main

import (
	"encoding/json"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/HughNian/nmid/pkg/model"
	wor "github.com/HughNian/nmid/pkg/worker"

	"github.com/vmihailenco/msgpack"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

func HealthCheck(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	type Params struct {
		Health string `json:"health"`
	}

	var params Params

	err := job.ShouldBind(&params)
	if err == nil {
		type Res struct {
			State   int
			Message string
		}

		var resData []byte

		if params.Health == "check" {
			var res = Res{
				200,
				"success",
			}

			resData, _ = json.Marshal(&res)
		} else {
			var res = Res{
				403,
				"forbidden",
			}

			resData, _ = json.Marshal(&res)
		}

		retStruct := model.GetRetStruct()
		retStruct.Msg = "ok"
		retStruct.Data = resData
		ret, err := msgpack.Marshal(retStruct)
		if nil != err {
			return []byte(``), err
		}

		resp.RetLen = uint32(len(ret))
		resp.Ret = ret

		return ret, nil
	} else {
		return nil, fmt.Errorf("invalid params")
	}
}

func ToUpper(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	type Params struct {
		Name string `json:"name"`
	}

	var params Params

	err := job.ShouldBind(&params)
	if err == nil {
		retStruct := model.GetRetStruct()
		retStruct.Code = 0
		retStruct.Msg = "ok"
		retStruct.Data = []byte(strings.ToUpper(params.Name))
		ret, err := msgpack.Marshal(retStruct)
		if nil != err {
			return []byte(``), err
		}

		resp.RetLen = uint32(len(ret))
		resp.Ret = ret

		return ret, nil
	} else {
		return nil, fmt.Errorf("invalid params")
	}
}

func main() {
	wname := "Worker1"

	var worker *wor.Worker
	var err error

	serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
	worker = wor.NewWorker().SetWorkerName(wname)
	err = worker.AddServer("tcp", serverAddr)
	if err != nil {
		log.Fatalln(err)
		worker.WorkerClose()
		return
	}

	worker.AddFunction("ToUpper", ToUpper)
	worker.AddFunction("HealthCheck", HealthCheck)

	if err = worker.WorkerReady(); err != nil {
		log.Fatalln(err)
		worker.WorkerClose()
		return
	}

	go worker.WorkerDo()

	quits := make(chan os.Signal, 1)
	signal.Notify(quits, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT /*syscall.SIGUSR1*/)
	switch <-quits {
	case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
		worker.WorkerClose()
	}
}
