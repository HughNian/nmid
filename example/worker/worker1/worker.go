// nmid worker
//
// author: niansong(hugh.nian@163.com)
package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/HughNian/nmid/pkg/model"
	wor "github.com/HughNian/nmid/pkg/worker"

	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
	"github.com/vmihailenco/msgpack"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

var discoverys = []string{"localhost:2379"}
var disUsername = "root"
var disPassword = "123456"

func ToUpper(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	if len(resp.ParamsMap) > 0 {
		name := resp.ParamsMap["name"].(string)

		retStruct := model.GetRetStruct()
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

	if len(resp.ParamsMap) > 0 {
		var orderType int
		orderSn := resp.ParamsMap["order_sn"].(string)
		switch resp.ParamsMap["order_type"].(type) {
		case int64:
			int64val := resp.ParamsMap["order_type"].(int64)
			orderType = int(int64val)
		case float64:
			float64val := resp.ParamsMap["order_type"].(float64)
			orderType = int(float64val)
		}

		retStruct := model.GetRetStruct()
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
	wname := "Worker1"

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
	//var skyReporterUrl = "192.168.64.6:30484"

	serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
	//worker = wor.NewWorker().SetWorkerName(wname).WithTrace(skyReporterUrl)
	worker = wor.NewWorker().SetWorkerName(wname)
	err = worker.AddServer("tcp", serverAddr)
	if err != nil {
		log.Fatalln(err)
		worker.WorkerClose()
		return
	}

	worker.AddFunction("ToUpper", ToUpper)
	worker.AddFunction("GetOrderInfo", GetOrderInfo)
	//register to discovery server
	worker.Register(wor.EtcdConfig{Addrs: discoverys, Username: disUsername, Password: disPassword})

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
