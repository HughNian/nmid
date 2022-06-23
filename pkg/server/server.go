package server

import (
	"log"
	"net"
	"sync"
)

type Server struct {
	mu    sync.Mutex
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

		ser.mu.Lock()
		c := ser.Cpool.NewConnect(ser, conn)
		go c.DoIO()
		ser.mu.Unlock()
	}
}
