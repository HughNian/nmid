<div align="center">
    <a href="http://www.niansong.top"><img src="https://raw.githubusercontent.com/HughNian/nmid/master/logo/nmidlogo.png" alt="nmid Logo" width="160"></a>
</div>

## nmid介绍

nmid意思为中场指挥官，足球场上的中场就是统领进攻防守的核心。咱们这里是服务程序的调度核心。是一个轻量级分布式微服务RPC框架。

1.pkg/server目录为nmid微服务调度服务端go实现，采用协程以及管道的异步通信，带有连接池，自有I/O通信协议，msgpack做通信数据格式。      

2.pkg/worker目录为nmid的工作端go实现，目前也有c语言实现，以及php扩展实现，可以实现golang, php, c等作为工作端，从而实现跨语言平台提供功能服务。             

3.pkg/client目录为nmid的客户端go实现，目前也有c语言实现，以及php扩展实现，可以实现golang, php, c等作为客户端，从而实现跨语言平台调用功能服务。      

4.cmd目录为demo运行目录。为go实现的客户端示例，调度服务端示例，客户端示例。目前调度服务端只有golang的实现。  

5.C语言版本：https://github.com/HughNian/nmid-c  

6.PHP扩展：https://github.com/HughNian/nmid-php-ext  

## 建议配置

```
cat /proc/version
Linux version 3.10.0-957.21.3.el7.x86_64 ...(centos7)

go version
go1.12.5 linux/amd64

gcc --version
gcc (GCC) 4.8.5 20150623 (Red Hat 4.8.5-36)

cmake --version
cmake version 3.11.4

```

## 编译安装步骤

```
git clone https://github.com/HughNian/nmid.git

1.client
cd nmid/run/client
make

2.server
cd nmid/run/server
make

3.worker
cd nmid/run/worker
make

```

## 使用

```cpp
客户端代码

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

	paramsName10 := []string{"key:Go was publicly announced in November 2009, and version 1.0 was released in March 2012. Go is widely used in production at Google and in many other organizations and open-source projects.Gopher mascot.In November 2016, the Go and Go Mono fonts were released by type designers Charles Bigelow and Kris Holmes specifically for use by the Go project. Go and Go Mono fonts are sans-serif and monospaced respectively. Both fonts adhere to WGL4 and were designed to be legible, with a large x-height and distinct letterforms, by conforming to the DIN 1450 standard.In April 2018, the original logo was replaced with a stylized GO slanting right with trailing streamlines. However, the Gopher mascot remained the same."}
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

```

```cpp
服务端代码

package main

import (
	ser "nmid-go/server"
)

func main() {
	var server *ser.Server
	server = ser.NewServer()

	if nil == server {
		return
	}

	server.ServerRun()
}

```

```cpp
工作端代码

package main

import (
	wor "nmid-go/worker"
	"fmt"
	"strings"
	"log"
	"github.com/vmihailenco/msgpack"
)

const SERVERHOST = "192.168.1.176"
const SERVERPORT = "6808"

//单个入参
func ToUpper(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	if resp.ParamsType == wor.PARAMS_TYPE_MUL {
		return []byte(``), fmt.Errorf("params num error")
	}

	name := resp.StrParams[0]

	retStruct := wor.GetRetStruct()
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

//多个入参
func GetOrderInfo(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	if resp.ParamsType != wor.PARAMS_TYPE_MUL {
		return []byte(``), fmt.Errorf("params num error")
	}

	orderSn, orderType := "", ""
	for _, v := range resp.StrParams {
		column := strings.Split(v, string(wor.PARAMS_SCOPE))
		switch column[0] {
			case "order_sn":
				orderSn = column[1]
			case "order_type":
				orderType = column[1]
		}
	}

	retStruct := wor.GetRetStruct()
	if orderSn == "MBO993889253" && orderType == "4" {
		retStruct.Msg  = "ok"
		retStruct.Data = []byte("good goods")
	} else {
		retStruct.Code = 100
		retStruct.Msg  = "params error"
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

func main() {
	var worker *wor.Worker
	var err error

	serverAddr := SERVERHOST + ":" + SERVERPORT
	worker = wor.NewWorker()
	err = worker.AddServer("tcp", serverAddr)
	if err != nil {
		log.Fatalln(err)
		worker.WorkerClose()
		return
	}

	worker.AddFunction("ToUpper", ToUpper)
	worker.AddFunction("GetOrderInfo", GetOrderInfo)

	if err = worker.WorkerReady(); err != nil {
		log.Fatalln(err)
		worker.WorkerClose()
		return
	}

	worker.WorkerDo()
}

```

## I/O的通信网络协议

- 包结构   

    1.包头：链接类型[uint32/4字节]+数据类型[uint32/4字节]+包体长度[uint32/4字节]   
    
        连接类型：0初始，1服务端server，2工作端worker，3客户端client。    
        数据类型: server数据请求，server数据返回，worker数据请求...。    
        包体长度：具体返回包体数据内容的总长度。
    
    2.包体：  
        
        (1)client => sever: 客户端请求服务端  
        包体长度 = UINT32_SIZE + HandleLen + UINT32_SIZE + ParamsLen   
                  方法名长度值空间+方法名长度空间+msgpack后参数长度值空间+msgpack后参数长度空间
                  
        包体包含 = 方法长度值+方法名称+msgpack后参数长度值+msgpack后的参数值   
        
        client请求参数数据：参数都为字符串数组，入参为  
        []string{"order_sn:MBO993889253", "order_type:4"}，xx:xxx形式，必须以:分隔
        类似key:value。
        
        
        
        (2)server => worker: 服务端请求工作端
        包体长度 = UINT32_SIZE + HandleLen + UINT32_SIZE + ParamsLen   
                  方法名长度值空间+方法名长度空间+msgpack后参数长度值空间+msgpack后参数长度空间
                          
        包体包含 = 方法长度值+msgpack后参数长度值+方法名称+msgpack后的参数值    
        
        server请求参数数据：参数都为字符串数组，入参为  
        []string{"order_sn:MBO993889253", "order_type:4"}，xx:xxx形式，必须以:分隔
        类似key:value。可以理解为server做了client的透传。    
        
        
        
        (3)worker => server: 工作端返回数据服务端   
        包体长度 = UINT32_SIZE + HandleLen + UINT32_SIZE + ParamsLen + UINT32_SIZE + RetLen   
                  方法名长度值空间+方法名长度空间+msgpack后参数长度值空间+msgpack后参数长度空间+msgpack后结果长度值空间+msgpack后结果长度空间
                                  
        包体包含 = 方法长度值+方法名称+msgpack后参数长度值+msgpack后的参数值+msgpack后结果长度值+msgpack后结果值   
        
        worker返回结果数据：返回数据为统一格式结构体
        type RetStruct struct {
            Code int
            Msg  string
            Data []byte
        }       
        
        
        
        (4)server => client: 服务端返回数据客户端
        包体长度 = UINT32_SIZE + HandleLen + UINT32_SIZE + ParamsLen + UINT32_SIZE + RetLen   
        方法名长度值空间+方法名长度空间+msgpack后参数长度值空间+msgpack后参数长度空间+msgpack后结果长度值空间+msgpack后结果长度空间
                                  
        包体包含 = 方法长度值+msgpack后参数长度值+msgpack后结果长度值+方法名称+msgpack后的参数值+msgpack后结果值   
        
        worker返回结果数据：返回数据为统一格式结构体
        type RetStruct struct {
            Code int
            Msg  string
            Data []byte
        }
        可以理解为server做了worker的透传。
        
        
## 交流博客

http://www.niansong.top