// nmid client
//
// author: niansong(hugh.nian@163.com)
//
//
package main

import (
	cli "nmid-go/client"
	"fmt"
	"log"
	"github.com/vmihailenco/msgpack"
	"os"
)

const SERVERHOST = "192.168.1.176"
const SERVERPORT = "6808"

func main() {
	var client *cli.Client
	var err error

	serverAddr := SERVERHOST + ":" + SERVERPORT
	client, err = cli.NewClient("tcp", serverAddr)
	if nil == client || err != nil {
		log.Println(err)
		return
	}
	defer client.Close()

	client.ErrHandler= func(e error) {
		log.Println(e)
		fmt.Println("client err here")
		//client.Close()
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
		}
	}


	//1 单个入参
	paramsName1 := []string{"name:niansong"}
	params1, err := msgpack.Marshal(&paramsName1)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("ToUpper", params1, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	//2 多个入参，参数中间以:分隔，xx:xxx
	paramsName2 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	params2, err := msgpack.Marshal(&paramsName2)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("GetOrderInfo", params2, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	paramsName3 := []string{"name:niansong", "pwd:123456"}
	params3, err := msgpack.Marshal(&paramsName3)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("PostInfo", params3, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	paramsName4 := []string{"order_sn:MBO993889253", "order_type:4"}
	params4, err := msgpack.Marshal(&paramsName4)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("GetOrderInfo", params4, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	paramsName5 := []string{"name:niansong", "pwd:123456"}
	params5, err := msgpack.Marshal(&paramsName5)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("PostInfo", params5, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	//2 多个入参，参数中间以:分隔，xx:xxx
	paramsName6 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	params6, err := msgpack.Marshal(&paramsName6)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("GetOrderInfo", params6, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	//2 多个入参，参数中间以:分隔，xx:xxx
	paramsName7 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	params7, err := msgpack.Marshal(&paramsName7)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("GetOrderInfo", params7, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	//2 多个入参，参数中间以:分隔，xx:xxx
	paramsName8 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	params8, err := msgpack.Marshal(&paramsName8)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("GetOrderInfo", params8, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	//2 多个入参，参数中间以:分隔，xx:xxx
	paramsName9 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	params9, err := msgpack.Marshal(&paramsName9)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("GetOrderInfo", params9, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	paramsName10 := []string{"key:Although China has never renounced the use of force to bring Taiwan under its control, it is rare for a top, serving military officer to so explicitly make the threat in a public setting. The comments are especially striking amid international opprobrium over China passing new national security legislation for Chinese-run Hong Kong"}
	params10, err := msgpack.Marshal(&paramsName10)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("ToUpper", params10, respHandler)
	if nil != err {
		fmt.Println(err)
	}
}