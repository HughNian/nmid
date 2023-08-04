// nmid server
//
// author: niansong(hugh.nian@163.com)
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/HughNian/nmid/pkg/conf"
	ser "github.com/HughNian/nmid/pkg/server"
)

var (
	sConfig = conf.GetConfig()
)

func main() {
	//pprof
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6061", nil))
	}()

	// godotenv.Load("./.env")

	//pyroscope, this is pyroscope push mode. also use pull mode better
	// profiler.Start(profiler.Config{
	// 	ApplicationName: "nmid.server",
	// 	ServerAddress:   "http://127.0.0.1:4040",
	// })

	rpcserver := ser.NewServer().SetSConfig(sConfig)
	if nil == rpcserver {
		return
	}

	//c, cancel := context.WithCancel(context.Background())
	_, cancel := context.WithCancel(context.Background())

	showLogo()

	//开启rpc tcp服务
	go rpcserver.ServerRun()
	//开启rpc http服务
	// go rpcserver.HttpServerRun()
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
	fmt.Println(logo)
}
