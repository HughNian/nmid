// nmid client
//
// author: niansong(hugh.nian@163.com)
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	cli "github.com/HughNian/nmid/pkg/client"
	"github.com/HughNian/nmid/pkg/model"

	"github.com/vmihailenco/msgpack"
)

const SERVERHOST = "127.0.0.1"
const SERVERPORT = "6808"

func main() {
	var client *cli.Client
	var err error

	serverAddr := SERVERHOST + ":" + SERVERPORT
	client, err = cli.NewClient("tcp", serverAddr).SetIoTimeOut(30 * time.Second).Start()
	if nil == client || err != nil {
		log.Println(err)
		return
	}
	defer client.Close()

	client.ErrHandler = func(e error) {
		if model.RESTIMEOUT == e {
			log.Println("time out here")
		} else {
			log.Println(e)
		}
		fmt.Println("client err here")
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
		}
	}

	paramsName1 := make(map[string]interface{})
	paramsName1["name"] = "nmid"
	params1, err := msgpack.Marshal(&paramsName1)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("ToUpper", params1, respHandler)
	if nil != err {
		fmt.Println(err)
	}
}
