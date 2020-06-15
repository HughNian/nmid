// nmid server
//
// author: niansong(hugh.nian@163.com)
//
//
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
