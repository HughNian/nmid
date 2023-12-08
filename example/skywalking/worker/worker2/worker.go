// nmid worker2
// this worker do client request anthor worker get result
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	cli "github.com/HughNian/nmid/pkg/client"
	"github.com/HughNian/nmid/pkg/model"
	wor "github.com/HughNian/nmid/pkg/worker"

	"github.com/vmihailenco/msgpack"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"
const SKYREPORTERURL = "192.168.10.176:11800" //skywalking的grpc地址

func ToUpper2(job wor.Job) (ret []byte, err error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	var name string
	if len(resp.ParamsMap) > 0 {
		name = resp.ParamsMap["name"].(string)
	}

	errHandler := func(e error) {
		if model.RESTIMEOUT == e {
			log.Println("time out here")
		} else {
			log.Println(e)
		}
	}

	respHandler := func(resp *cli.Response) {
		if resp.DataType == model.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				log.Println("ret empty")
				err = errors.New("ret empty")
				return
			}

			var cretStruct model.RetStruct
			uerr := msgpack.Unmarshal(resp.Ret, &cretStruct)
			if nil != uerr {
				log.Fatalln(uerr)
				err = uerr
				return
			}

			if cretStruct.Code != 0 {
				log.Println(cretStruct.Msg)
				err = errors.New(cretStruct.Msg)
				return
			}
			fmt.Println(string(cretStruct.Data))

			wretStruct := model.GetRetStruct()
			wretStruct.Msg = "ok"
			wretStruct.Data = cretStruct.Data
			ret, err = msgpack.Marshal(wretStruct)

			resp.RetLen = uint32(len(ret))
			resp.Ret = ret
		}
	}

	callAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
	funcName := "ToUpper"
	paramsName1 := make(map[string]interface{})
	paramsName1["name"] = name
	job.ClientCall(callAddr, funcName, paramsName1, respHandler, errHandler)

	return
}

func main() {
	wname := "Worker2"

	var worker *wor.Worker
	var err error

	serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
	worker = wor.NewWorker().SetWorkerName(wname).WithTrace(SKYREPORTERURL)
	err = worker.AddServer("tcp", serverAddr)
	if err != nil {
		log.Fatalln(err)
		worker.WorkerClose()
		return
	}

	worker.AddFunction("ToUpper2", ToUpper2)

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
