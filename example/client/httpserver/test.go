package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/vmihailenco/msgpack"

	_ "net/http/pprof"

	cli "github.com/HughNian/nmid/pkg/client"

	"github.com/buaazp/fasthttprouter"
	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
	"github.com/valyala/fasthttp"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

var once sync.Once
var client *cli.Client
var err error

var discoverys = []string{"localhost:2379"}
var consumer *cli.Consumer

func getClient() *cli.Client {
	serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
	client, err := cli.NewClient("tcp", serverAddr).SetIoTimeOut(30 * time.Second).Start()
	if nil == client || err != nil {
		logger.Error(err)
	}

	return client
}

func discovery(funcName string) *cli.Client {
	client := consumer.Discovery(funcName)
	if client != nil {
		client, err := client.SetIoTimeOut(30 * time.Second).Start()
		if nil == client || err != nil {
			logger.Error(err)
		}
	} else {
		client = getClient()
	}

	return client
}

func Test(ctx *fasthttp.RequestCtx) {
	funcName := "ToUpper"

	//client := getClient()
	client := discovery(funcName)
	defer client.Close()

	if nil == client {
		fmt.Fprint(ctx, "nmid client error")
		return
	}

	client.SetParamsType(model.PARAMS_TYPE_JSON)

	client.ErrHandler = func(e error) {
		if model.RESTIMEOUT == e {
			logger.Warn("time out here")
		} else {
			logger.Error(e)
		}

		fmt.Fprint(ctx, e.Error())
	}

	respHandler := func(resp *cli.Response) {
		if resp.DataType == model.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				logger.Info("ret empty")
				return
			}

			var retStruct model.RetStruct
			err := msgpack.Unmarshal(resp.Ret, &retStruct)
			if nil != err {
				log.Fatalln(err)
				return
			}

			if retStruct.Code != 0 {
				log.Println(retStruct.Msg)
				return
			}

			fmt.Println(string(retStruct.Data))

			fmt.Fprint(ctx, string(retStruct.Data))
		}
	}

	paramsName := make(map[string]interface{})
	paramsName["name"] = "niansong1"
	//params, err := msgpack.Marshal(&paramsName)
	params1, err := json.Marshal(&paramsName)
	if err != nil {
		logger.Fatal("params msgpack error:", err)
	}
	err = client.Do(funcName, params1, respHandler)
	if nil != err {
		logger.Error(`do err`, err)
	}
}

func main() {
	//pprof
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	//pyroscope, this is pyroscope push mode. also use pull mode better
	profiler.Start(profiler.Config{
		ApplicationName: "nmid.httpclient",
		ServerAddress:   "http://127.0.0.1:4040",
	})

	consumer = &cli.Consumer{
		EtcdAddrs: discoverys,
	}
	consumer.EtcdCli = consumer.EtcdClient()
	consumer.EtcdWatch()

	router := fasthttprouter.New()
	router.GET("/test", Test)
	err := fasthttp.ListenAndServe(":5981", router.Handler)
	fmt.Println(`err info:`, err)
}
