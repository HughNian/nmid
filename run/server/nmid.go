// nmid server
//
// author: niansong(hugh.nian@163.com)
//
//
package main

import (
	ser "nmid-go/server"
)

var (
	confstruct ser.ServerConfig
	conf = confstruct.GetConfig()
)

func main() {
	var server *ser.Server
	server = ser.NewServer(conf.NETWORK, conf.HOST, conf.PORT)

	if nil == server {
		return
	}

	server.ServerRun()
}
