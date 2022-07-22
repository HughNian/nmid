//nmid worker2
//this worker do client request anthor worker get result
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	cli "nmid-v2/pkg/client"
	"nmid-v2/pkg/conf"
	wor "nmid-v2/pkg/worker"
	"sync"

	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
	"github.com/vmihailenco/msgpack"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

var once sync.Once
var client *cli.Client
var err error

func getClient() *cli.Client {
	if nil == client {
		serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
		client, err = cli.NewClient("tcp", serverAddr)
		if nil == client || err != nil {
			log.Println(err)
		}
	}

	return client
}

func ToUpper2(job wor.Job) (ret []byte, err error) {
	client := getClient()

	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	var name string
	if len(resp.ParamsMap) > 0 {
		name = resp.ParamsMap["name"].(string)
	}

	client.ErrHandler = func(e error) {
		if conf.RESTIMEOUT == e {
			log.Println("time out here")
		} else {
			log.Println(e)
		}
	}

	respHandler := func(resp *cli.Response) {
		if resp.DataType == conf.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				log.Println("ret empty")
				err = errors.New("ret empty")
				return
			}

			var cretStruct conf.RetStruct
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

			wretStruct := wor.GetRetStruct()
			wretStruct.Msg = "ok"
			wretStruct.Data = cretStruct.Data
			ret, err = msgpack.Marshal(wretStruct)

			resp.RetLen = uint32(len(ret))
			resp.Ret = ret
		}
	}

	paramsName1 := make(map[string]string)
	paramsName1["name"] = name
	params1, err := msgpack.Marshal(&paramsName1)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
	}
	err = client.Do("ToUpper", params1, respHandler)
	if nil != err {
		fmt.Println(`--do err--`, err)
	}

	return
}

func main() {
	//pprof
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6063", nil))
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

	worker.AddFunction("ToUpper2", ToUpper2)

	if err = worker.WorkerReady(); err != nil {
		log.Fatalln(err)
		worker.WorkerClose()
		return
	}

	worker.WorkerDo()
}
