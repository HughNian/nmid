// nmid server
//
// author: niansong(hugh.nian@163.com)
//
//
package main

import (
	"context"
	ser "nmid-v2/pkg/server"
	"os"
	"os/signal"
	"syscall"
)

var (
	confstruct ser.ServerConfig
	conf       = confstruct.GetConfig()
)

func main() {
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
