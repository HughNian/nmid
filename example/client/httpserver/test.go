package main

import (
	"encoding/json"
	"fmt"
	"github.com/HughNian/nmid/pkg/model"
	"log"
	"net/http"
	"sync"

	cli "github.com/HughNian/nmid/pkg/client"
	_ "net/http/pprof"

	"github.com/buaazp/fasthttprouter"
	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
	"github.com/valyala/fasthttp"
	"github.com/vmihailenco/msgpack"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

var once sync.Once
var client *cli.Client
var err error

//单实列连接，适合长连接
func getClient() *cli.Client {
	once.Do(func() {
		serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
		client, err = cli.NewClient("tcp", serverAddr)
		if nil == client || err != nil {
			log.Println(err)
		}
	})

	return client
}

func Test(ctx *fasthttp.RequestCtx) {
	client := getClient()
	client.SetParamsType(model.PARAMS_TYPE_JSON)

	client.ErrHandler = func(e error) {
		if model.RESTIMEOUT == e {
			log.Println("time out here")
		} else {
			log.Println(e)
		}

		fmt.Fprint(ctx, e.Error())
	}

	respHandler := func(resp *cli.Response) {
		if resp.DataType == model.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				log.Println("ret empty")
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

	respHandler2 := func(resp *cli.Response) {
		if resp.DataType == model.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				log.Println("ret empty")
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

	respHandler3 := func(resp *cli.Response) {
		if resp.DataType == model.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				log.Println("ret empty")
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

	paramsName1 := make(map[string]interface{})
	paramsName1["name"] = "niansong"
	//params1, err := msgpack.Marshal(&paramsName1)
	params1, err := json.Marshal(&paramsName1)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
	}
	err = client.Do("ToUpper", params1, respHandler)
	if nil != err {
		fmt.Println(`--do err--`, err)
	}

	paramsName2 := make(map[string]interface{})
	paramsName2["name"] = "niansong2"
	//params2, err := msgpack.Marshal(&paramsName2)
	params2, err := json.Marshal(&paramsName2)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
	}
	err = client.Do("ToUpper2", params2, respHandler2)
	if nil != err {
		fmt.Println(`--do2 err--`, err)
	}

	paramsName3 := make(map[string]interface{})
	paramsName3["order_sn"] = "MBO993889253"
	paramsName3["order_type"] = 4
	//params3, err := msgpack.Marshal(&paramsName3)
	params3, err := json.Marshal(&paramsName3)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
	}
	err = client.Do("GetOrderInfo", params3, respHandler3)
	if nil != err {
		fmt.Println(`--do3 err--`, err)
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

	router := fasthttprouter.New()
	router.GET("/test", Test)
	err := fasthttp.ListenAndServe(":5981", router.Handler)
	fmt.Println(`err info:`, err)
}
