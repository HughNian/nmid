// nmid server
//
// author: niansong(hugh.nian@163.com)
package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/HughNian/nmid/pkg/conf"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/metric"
	ser "github.com/HughNian/nmid/pkg/server"
)

var (
	Version string
	GitHash string
)

func main() {
	//pprof
	// go func() {
	// 	log.Println(http.ListenAndServe("0.0.0.0:6061", nil))
	// }()

	// pyroscope, this is pyroscope push mode. also use pull mode better
	// profiler.Start(profiler.Config{
	// 	ApplicationName: "nmid.server",
	// 	ServerAddress:   "http://192.168.10.176:4040",
	// })

	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("nmid server crash error %v", err)
		}
	}()

	rpcserver := ser.NewServer().SetSConfig()
	if nil == rpcserver {
		return
	}
	rpcserver.SetStartUp()

	//c, cancel := context.WithCancel(context.Background())
	_, cancel := context.WithCancel(context.Background())

	showLogo()

	//开启rpc tcp服务
	go rpcserver.ServerRun()

	//开启rpc http服务
	if len(conf.GetConfig().RpcServer.HTTPPORT) != 0 {
		go rpcserver.HttpServerRun()
	}

	//开启sidecar
	// scCtx, scCancel := context.WithCancel(c)
	// sidecar.NewScServer(scCtx, sConfig).StartScServer()

	quits := make(chan os.Signal, 1)
	signal.Notify(quits, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT /*syscall.SIGUSR1*/)
	switch <-quits {
	case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
		cancel()
		// case syscall.SIGUSR1:
		//scCancel()
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	rpcserver.ServerClose(wg)

	if conf.GetConfig().Prometheus.Enable {
		wg.Add(1)
		metric.DoCloseListenerWithWg(wg)
	}

	wg.Wait()
	os.Exit(0)
}

func showLogo() {
	logo := `
                    _     __
   ____  ____ ___  (_)___/ /
  / __ \/ __ \__ \/ / __  / 
 / / / / / / / / / / /_/ /  
/_/ /_/_/ /_/ /_/_/\__,_/
`
	logoVersion := fmt.Sprintf("%s\nVersion:%s\n", logo, Version)
	fmt.Println(logoVersion)
}
