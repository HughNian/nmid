// nmid worker
//
// author: niansong(hugh.nian@163.com)
//
//
package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"nmid-v2/pkg/conf"
	wor "nmid-v2/pkg/worker"
	"strings"

	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
	"github.com/vmihailenco/msgpack"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

func ToUpper(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	if resp.ParamsType == conf.PARAMS_TYPE_MSGPACK && len(resp.ParamsMap) > 0 {
		name := resp.ParamsMap["name"].(string)

		retStruct := wor.GetRetStruct()
		retStruct.Msg = "ok"
		retStruct.Data = []byte(strings.ToUpper(name))
		ret, err := msgpack.Marshal(retStruct)
		if nil != err {
			return []byte(``), err
		}

		resp.RetLen = uint32(len(ret))
		resp.Ret = ret

		return ret, nil
	}

	return nil, fmt.Errorf("response data error")
}

func GetOrderInfo(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	if resp.ParamsType == conf.PARAMS_TYPE_MSGPACK && len(resp.ParamsMap) > 0 {
		orderSn := resp.ParamsMap["order_sn"].(string)
		orderType := resp.ParamsMap["order_type"].(int64)

		retStruct := wor.GetRetStruct()
		if orderSn == "MBO993889253" && orderType == 4 {
			retStruct.Msg = "ok"
			retStruct.Data = []byte("good goods")
		} else {
			retStruct.Code = 100
			retStruct.Msg = "params error"
			retStruct.Data = []byte(``)
		}

		ret, err := msgpack.Marshal(retStruct)
		if nil != err {
			return []byte(``), err
		}

		resp.RetLen = uint32(len(ret))
		resp.Ret = ret

		return ret, nil
	}

	return nil, fmt.Errorf("response data error")
}

func main() {
	//pprof
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6062", nil))
	}()

	//pyroscope, this is pyroscope push mode. also use pull mode better
	profiler.Start(profiler.Config{
		ApplicationName: "nmid.worker",
		ServerAddress:   "http://127.0.0.1:4040",
	})

	var worker *wor.Worker
	var err error

	serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
	worker = wor.NewWorker()
	err = worker.AddServer("tcp", serverAddr)
	if err != nil {
		log.Fatalln(err)
		worker.WorkerClose()
		return
	}

	worker.AddFunction("ToUpper", ToUpper)
	worker.AddFunction("GetOrderInfo", GetOrderInfo)

	if err = worker.WorkerReady(); err != nil {
		log.Fatalln(err)
		worker.WorkerClose()
		return
	}

	worker.WorkerDo()
}
