package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"

	_ "net/http/pprof"

	cli "github.com/HughNian/nmid/pkg/client"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"github.com/vmihailenco/msgpack"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

var client *cli.Client
var err error

func getClient() *cli.Client {
	serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
	client, err := cli.NewClient("tcp", serverAddr).SetIoTimeOut(30 * time.Second).Start()
	if nil == client || err != nil {
		logger.Error(err)
	}

	return client
}

func Test(ctx *fasthttp.RequestCtx) {
	client := getClient()
	defer client.Close()

	if nil == client {
		fmt.Fprint(ctx, "nmid client error")
		return
	}

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
}

func CheckToken(ctx *fasthttp.RequestCtx) {
	param := ctx.QueryArgs().Peek("token")
	signature, err := base64.StdEncoding.DecodeString(string(param))
	if err != nil {
		logger.Error("signature base64 decode error %s", err.Error())
		fmt.Fprint(ctx, "signature base64 decode error")
	}
	token := string(signature)
	if token == "" {
		token = "NNxSfqb2r2TLltwzpUeyqf0+zzXRD9ga2jjtwauHWysdMV67WlZlrkDdtfOrFdVrrvxLZMrbiTZVvTLx6IDfC9UpwgC71n2iRAXf6Y49uQhEKZez4KzSlNvpQ4huPT9zl+9MBqHoLYdfH/V42mgYKg=="
	}

	ret := true
	client := getClient()
	defer client.Close()

	if nil == client {
		logger.Error("nmid client get error")
		fmt.Fprint(ctx, "nmid client get error")
	}

	client.ErrHandler = func(e error) {
		if model.RESTIMEOUT == e {
			logger.Error("time out here --- token: %s", token)
			fmt.Fprint(ctx, "time out here")
			return
		} else {
			logger.Error(e.Error())
			fmt.Fprint(ctx, e.Error())
			return
		}
	}

	respHandler := func(resp *cli.Response) {
		if resp.DataType == model.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				ret = false
				logger.Error("ret empty")
				return
			}

			var retStruct model.RetStruct
			err := msgpack.Unmarshal(resp.Ret, &retStruct)
			if nil != err {
				ret = false
				return
			}

			if retStruct.Code != 0 {
				logger.Info("%s --- token: %s", retStruct.Msg, token)
				ret = false
				return
			}

			ret = true
		}
	}

	paramsName := make(map[string]interface{})
	paramsName["token"] = token
	params, err := msgpack.Marshal(&paramsName)
	if err != nil {
		logger.Error("params msgpack marshal error: %s", err.Error())

		fmt.Fprint(ctx, "params msgpack marshal error")
	}
	err = client.Do("gateway/FuncCheckToken", params, respHandler)
	if nil != err {
		logger.Error(`do err %s`, err.Error())
	}

	fmt.Println(ret)

	fmt.Fprint(ctx, ret)
}

func main() {
	//pprof
	// go func() {
	// 	log.Println(http.ListenAndServe("0.0.0.0:6064", nil))
	// }()

	//pyroscope, this is pyroscope push mode. also use pull mode better
	// profiler.Start(profiler.Config{
	// 	ApplicationName: "nmid.httpclient",
	// 	ServerAddress:   "http://192.168.10.174:4040",
	// })

	router := fasthttprouter.New()
	router.GET("/test", CheckToken)
	err := fasthttp.ListenAndServe(":5982", router.Handler)
	fmt.Println(`err info:`, err)
}
