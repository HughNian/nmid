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

7.支持http请求nmid服务

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

import (
	"fmt"
	"log"
	"net/http"
	"nmid-v2/pkg/conf"
	"sync"

	_ "net/http/pprof"
	cli "nmid-v2/pkg/client"

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
    //client.SetParamsType(conf.PARAMS_TYPE_JSON)
    
	client.ErrHandler = func(e error) {
		if conf.RESTIMEOUT == e {
			log.Println("time out here")
		} else {
			log.Println(e)
		}

		fmt.Fprint(ctx, e.Error())
	}

	respHandler := func(resp *cli.Response) {
		if resp.DataType == conf.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				log.Println("ret empty")
				return
			}

			var retStruct conf.RetStruct
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
		if resp.DataType == conf.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				log.Println("ret empty")
				return
			}

			var retStruct conf.RetStruct
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
		if resp.DataType == conf.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				log.Println("ret empty")
				return
			}

			var retStruct conf.RetStruct
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
	params1, err := msgpack.Marshal(&paramsName1)
	//params1, err := json.Marshal(&paramsName1)
	if err != nil {
		log.Fatalln("params msgpack error:", err)
	}
	err = client.Do("ToUpper", params1, respHandler)
	if nil != err {
		fmt.Println(`--do err--`, err)
	}

	paramsName2 := make(map[string]interface{})
	paramsName2["name"] = "niansong2"
	params2, err := msgpack.Marshal(&paramsName2)
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
	params3, err := msgpack.Marshal(&paramsName3)
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

```

```cpp
服务端代码

package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"nmid-v2/pkg/conf"
	ser "nmid-v2/pkg/server"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
)

var (
	sConfig = conf.GetConfig()
)

func main() {
	//pprof
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6061", nil))
	}()

	//pyroscope, this is pyroscope push mode. also use pull mode better
	profiler.Start(profiler.Config{
		ApplicationName: "nmid.server",
		ServerAddress:   "http://127.0.0.1:4040",
	})

	server := ser.NewServer().SetSConfig(sConfig)
	if nil == server {
		return
	}

	_, cancel := context.WithCancel(context.Background())

	//开启tcp服务
	go server.ServerRun()
	//开启http服务
	go server.HttpServerRun()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	cancel()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	server.ServerClose(wg)
	wg.Wait()
	os.Exit(0)
}

```

```cpp
工作端代码

package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"nmid-v2/pkg/conf"
	wor "nmid-v2/pkg/worker"
	"strconv"
	"strings"

	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
	"github.com/vmihailenco/msgpack"
)

const NMIDSERVERHOST = "127.0.0.1"
const NMIDSERVERPORT = "6808"

func ToUpper(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	if len(resp.ParamsMap) > 0 {
		name := resp.ParamsMap["name"].(string)

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

	return nil, fmt.Errorf("response data error")
}

func GetOrderInfo(job wor.Job) ([]byte, error) {
	resp := job.GetResponse()
	if nil == resp {
		return []byte(``), fmt.Errorf("response data error")
	}

	if  len(resp.ParamsMap) > 0 {
	    var orderType int
		orderSn := resp.ParamsMap["order_sn"].(string)
		switch resp.ParamsMap["order_type"].(type) {
		case int64:
			int64val := resp.ParamsMap["order_type"].(int64)
			orderType = int(int64val)
		case float64:
			float64val := resp.ParamsMap["order_type"].(float64)
			orderType = int(float64val)
		}

		retStruct := wor.GetRetStruct()
		if orderSn == "MBO993889253" && orderType == 4 {
			retStruct.Msg = "ok"
			retStruct.Data = []byte("good goods")
		} else {
			retStruct.Code = 100
			retStruct.Msg = "params error"
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

	return nil, fmt.Errorf("response data error")
}

func main() {
	//pprof
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6062", nil))
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

```cpp
http客户端请求

func main() {
	args := make(map[string]interface{})
	args["name"] = "testtestcontent"
	data, _ := json.Marshal(args)
	req, err := http.NewRequest("POST", "http://127.0.0.1:6809/", bytes.NewReader(data))
	if err != nil {
		log.Fatal("failed to create request: ", err)
		return
	}

	h := req.Header
	h.Set(conf.NRequestType, conf.HTTPDOWORK)
	h.Set(conf.NParamsType, conf.PARAMSTYPEMSGPACK)
	h.Set(conf.NParamsHandleType, conf.PARAMSHANDLETYPEENCODE)
	h.Set(conf.NFunctionName, "ToUpper")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("failed to call: ", err)
	}
	defer res.Body.Close()

	// handle http response
	replyData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("failed to read response: ", err)
	}

	log.Println("ret data", string(replyData))
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