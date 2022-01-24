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
	wor "nmid-v2/pkg/worker"
	"strconv"
	"strings"

	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
	"github.com/vmihailenco/msgpack"
)

const SERVERHOST = "127.0.0.1"
const SERVERPORT = "6808"

//单个入参
func ToUpper(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	if resp.ParamsType == wor.PARAMS_TYPE_MUL {
		return []byte(``), fmt.Errorf("params num error")
	}

	name := resp.StrParams[0]

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

//多个入参
func GetOrderInfo(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	if resp.ParamsType != wor.PARAMS_TYPE_MUL {
		return []byte(``), fmt.Errorf("params num error")
	}

	orderSn, orderType := "", ""
	for _, v := range resp.StrParams {
		column := strings.Split(v, strconv.Itoa(wor.PARAMS_SCOPE))
		switch column[0] {
		case "order_sn":
			orderSn = column[1]
		case "order_type":
			orderType = column[1]
		}
	}

	retStruct := wor.GetRetStruct()
	if orderSn == "MBO993889253" && orderType == "4" {
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

func main() {
	//pprof
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6062", nil))
	}()

	//pyroscope
	profiler.Start(profiler.Config{
		ApplicationName: "nmid.worker",
		ServerAddress:   "http://127.0.0.1:4040",
	})

	var worker *wor.Worker
	var err error

	serverAddr := SERVERHOST + ":" + SERVERPORT
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
