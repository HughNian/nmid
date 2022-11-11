// nmid client
//
// author: niansong(hugh.nian@163.com)
package main

import (
	"fmt"
	cli "github.com/HughNian/nmid/pkg/client"
	"github.com/HughNian/nmid/pkg/model"
	"log"
	"os"

	"github.com/vmihailenco/msgpack"
)

const SERVERHOST = "127.0.0.1"
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
	//defer client.Close()

	client.ErrHandler = func(e error) {
		if model.RESTIMEOUT == e {
			log.Println("time out here")
		} else {
			log.Println(e)
		}
		fmt.Println("client err here")
		//client.Close()
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
	paramsName1["name"] = "niansong"
	params1, err := msgpack.Marshal(&paramsName1)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
		os.Exit(1)
	}
	err = client.Do("ToUpper", params1, respHandler)
	if nil != err {
		fmt.Println(err)
	}

	//cellphone := "13913873440"
	//obdnum := "531907250005"
	//carnum := "浙A9P733"
	//paramsPhone := []string{"phone:" + cellphone, "odbnum:" + obdnum, "carnum:" + carnum}
	//phone, err := msgpack.Marshal(&paramsPhone)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("sendObdPullOutSms", phone, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}

	//2 多个入参，参数中间以:分隔，xx:xxx
	//paramsName2 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	//params2, err := msgpack.Marshal(&paramsName2)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("GetOrderInfo", params2, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}
	//
	//paramsName3 := []string{"name:niansong", "pwd:123456"}
	//params3, err := msgpack.Marshal(&paramsName3)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("PostInfo", params3, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}
	//
	//paramsName4 := []string{"order_sn:MBO993889253", "order_type:4"}
	//params4, err := msgpack.Marshal(&paramsName4)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("GetOrderInfo", params4, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}
	//
	//paramsName5 := []string{"name:niansong", "pwd:123456"}
	//params5, err := msgpack.Marshal(&paramsName5)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("PostInfo", params5, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}
	//
	////2 多个入参，参数中间以:分隔，xx:xxx
	//paramsName6 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	//params6, err := msgpack.Marshal(&paramsName6)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("GetOrderInfo", params6, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}
	//
	////2 多个入参，参数中间以:分隔，xx:xxx
	//paramsName7 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	//params7, err := msgpack.Marshal(&paramsName7)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("GetOrderInfo", params7, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}
	//
	////2 多个入参，参数中间以:分隔，xx:xxx
	//paramsName8 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	//params8, err := msgpack.Marshal(&paramsName8)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("GetOrderInfo", params8, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}
	//
	////2 多个入参，参数中间以:分隔，xx:xxx
	//paramsName9 := []string{"order_sn:MBO993889253", "order_type:4", "fenxiao:2253", "open_id:all", "order_status:1"}
	//params9, err := msgpack.Marshal(&paramsName9)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("GetOrderInfo", params9, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}
	//
	//paramsName10 := []string{"key:Go was publicly announced in November 2009, and version 1.0 was released in March 2012. Go is widely used in production at Google and in many other organizations and open-source projects.Gopher mascot.In November 2016, the Go and Go Mono fonts were released by type designers Charles Bigelow and Kris Holmes specifically for use by the Go project. Go and Go Mono fonts are sans-serif and monospaced respectively. Both fonts adhere to WGL4 and were designed to be legible, with a large x-height and distinct letterforms, by conforming to the DIN 1450 standard.In April 2018, the original logo was replaced with a stylized GO slanting right with trailing streamlines. However, the Gopher mascot remained the same."}
	//params10, err := msgpack.Marshal(&paramsName10)
	//if err != nil {
	//	log.Fatalln("params msgpack error:", err)
	//	os.Exit(1)
	//}
	//err = client.Do("ToUpper", params10, respHandler)
	//if nil != err {
	//	fmt.Println(err)
	//}
}
