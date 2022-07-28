// nmid server
//
// author: niansong(hugh.nian@163.com)
//
//
package main

import (
	"context"
	"fmt"
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

	showLogo()

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

func showLogo() {
	logo := `
                          /$$      /$$
                         |__/     | $$
/$$$$$$$  /$$$$$$/$$$$   /$$  /$$$$$$$
| $$__  $$| $$_  $$_  $$| $$ /$$__  $$
| $$  \ $$| $$ \ $$ \ $$| $$| $$  | $$
| $$  | $$| $$ | $$ | $$| $$| $$  | $$
| $$  | $$| $$ | $$ | $$| $$|  $$$$$$$
|__/  |__/|__/ |__/ |__/|__/ \_______/
`
	fmt.Println(logo)
}
