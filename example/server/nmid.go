// nmid server
//
// author: niansong(hugh.nian@163.com)
//
//
package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	ser "nmid-v2/pkg/server"
	"os"
	"os/signal"
	"syscall"

	"github.com/pyroscope-io/pyroscope/pkg/agent/profiler"
)

var (
	sConfig = ser.GetConfig()
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

	server := ser.NewServer(sConfig.Server.NETWORK, sConfig.Server.HOST, sConfig.Server.PORT).SetHttpPort(sConfig.Server.HTTPPORT)
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
	os.Exit(0)
}
