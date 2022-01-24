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
	confstruct ser.ServerConfig
	conf       = confstruct.GetConfig()
)

func main() {
	//pprof
	go func() {
		log.Println(http.ListenAndServe("0.0.0.0:6061", nil))
	}()

	//pyroscope
	profiler.Start(profiler.Config{
		ApplicationName: "nmid.server",
		ServerAddress:   "http://127.0.0.1:4040",
	})

	server := ser.NewServer(conf.NETWORK, conf.HOST, conf.PORT)
	if nil == server {
		return
	}

	_, cancel := context.WithCancel(context.Background())

	go server.ServerRun()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	cancel()
	os.Exit(0)
}
