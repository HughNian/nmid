package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/HughNian/nmid/pkg/direct"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/vmihailenco/msgpack"

	_ "net/http/pprof"

	cli "github.com/HughNian/nmid/pkg/client"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

var client *cli.Client
var err error
var directPool chan *direct.Client
var directOnce bool

func getClient() *cli.Client {
	serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
	client, err := cli.NewClient("tcp", serverAddr).SetIoTimeOut(1 * time.Second).Start()
	if nil == client || err != nil {
		logger.Error(err)
	}

	return client
}

func getDirectClient() *direct.Client {
	if !directOnce {
		directOnce = true
		addr := os.Getenv("DIRECT_ADDR")
		if addr == "" {
			addr = "127.0.0.1:7900"
		}
		directPool = make(chan *direct.Client, 200)
		for i := 0; i < cap(directPool); i++ {
			c := direct.NewClient("tcp", addr)
			c.Timeout = 1500 * time.Millisecond
			directPool <- c
		}
	}
	select {
	case c := <-directPool:
		return c
	default:
		return nil
	}
}

func putDirectClient(c *direct.Client) {
	if c == nil {
		return
	}
	select {
	case directPool <- c:
	default:
		_ = c.Close()
	}
}

func Test(ctx *fasthttp.RequestCtx) {
	if os.Getenv("NMID_MODE") == "direct" {
		c := getDirectClient()
		if c == nil {
			ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
			fmt.Fprint(ctx, "ER")
			return
		}
		defer putDirectClient(c)
		out, err := c.Call("ToUpper", []byte(`{"name":"nmid"}`))
		if err != nil {
			_ = c.Close()
			ctx.SetStatusCode(fasthttp.StatusRequestTimeout)
			fmt.Fprint(ctx, "ER")
			return
		}
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.Write(out)
		return
	}

	funcName := "ToUpper"

	client := getClient()
	defer client.Close()

	if nil == client {
		fmt.Fprint(ctx, "nmid client error")
		return
	}

	client.SetParamsType(model.PARAMS_TYPE_JSON) //以json格式传输参数

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
	paramsName["name"] = "nmid"
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
	router := fasthttprouter.New()
	router.GET("/test", Test)
	err := fasthttp.ListenAndServe(":5981", router.Handler)
	fmt.Println(`err info:`, err)
}
