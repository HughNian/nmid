package main

import (
	"fmt"
	"log"

	cli "nmid-v2/pkg/client"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"github.com/vmihailenco/msgpack"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

var client *cli.Client
var err error

// func init() {
// 	serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
// 	client, err = cli.NewClient("tcp", serverAddr)
// 	if nil == client || err != nil {
// 		log.Println(err)
// 		return
// 	}
// 	// defer client.Close()
// }

func Test(ctx *fasthttp.RequestCtx) {
	serverAddr := NMIDSERVERHOST + ":" + NMIDSERVERPORT
	client, err = cli.NewClient("tcp", serverAddr)
	if nil == client || err != nil {
		log.Println(err)
		return
	}
	// defer client.Close()

	client.ErrHandler = func(e error) {
		if cli.RESTIMEOUT == e {
			log.Println("time out here")
		} else {
			log.Println(e)
		}
		fmt.Println("client err here")

		fmt.Fprint(ctx, `time out`)
	}

	respHandler := func(resp *cli.Response) {
		if resp.DataType == cli.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				log.Println("ret empty")
				return
			}

			var retStruct cli.RetStruct
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

	paramsName1 := []string{"name:niansong"}
	params1, err := msgpack.Marshal(&paramsName1)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
	}
	err = client.Do("ToUpper", params1, respHandler)
	if nil != err {
		fmt.Println(`--do err--`, err)
	}
}

func main() {
	router := fasthttprouter.New()
	router.GET("/test", Test)
	err := fasthttp.ListenAndServe(":5981", router.Handler)
	fmt.Println(`err info:`, err)
}
