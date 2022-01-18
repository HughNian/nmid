package server

import (
	"log"
	"net"
)

type Server struct {
	Host  string
	Port  string
	Net   string
	Cpool *ConnectPool
	Funcs *FuncMap
}

func NewServer(net string, host string, port string) (ser *Server) {
	ser = &Server{
		Host:  host,
		Port:  port,
		Net:   net,
		Cpool: NewConnectPool(),
		Funcs: NewFuncMap(),
	}
	return
}

func (ser *Server) ServerRun() {
	var address string = ser.Host + ":" + ser.Port
	listen, err := net.Listen(ser.Net, address)
	if err != nil {
		log.Fatalln(err)
		panic(err)
	}

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
			continue
		}

		c := ser.Cpool.NewConnect(ser, conn)
		go c.DoIO()
	}
}
