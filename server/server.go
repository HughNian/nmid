package server

import (
	"net"
	"log"
)

type Server struct {
	Host   string
	Port   string
	Net    string
	Cpool  *ConnectPool
	Funcs  *FuncMap
}

func NewServer() (ser *Server) {
	ser = &Server {
		Host  : HOST,
		Port  : PORT,
		Net   : NETWORK,
		Cpool : NewConnectPool(),
		Funcs : NewFuncMap(),
	}
	return
}

func (ser *Server) ServerRun() {
	var address string

	address = ser.Host + ":" + ser.Port
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