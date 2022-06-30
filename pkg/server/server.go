package server

import (
	"crypto/tls"
	"log"
	"net/http"
	"sync"
)

type Server struct {
	sync.Mutex

	Host              string
	Port              string
	HttpPort          string
	Net               string
	Cpool             *ConnectPool
	Funcs             *FuncMap
	TlsConfig         *tls.Config
	HTTPServerGateway *http.Server
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

func (ser *Server) SetHttpPort(HttpPort string) {
	ser.HttpPort = HttpPort
}

func (ser *Server) ServerRun() {
	var address string = ser.Host + ":" + ser.Port
	listen, err := ser.MakeListener(ser.Net, address)
	if err != nil {
		log.Fatalln(err)
		panic(err)
	}

	if len(ser.HttpPort) > 0 {
		ser.NewHTTPAPIGateway("http")
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
