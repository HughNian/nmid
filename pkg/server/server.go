package server

import (
	"crypto/tls"
	"errors"
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

func (ser *Server) SetHttpPort(HttpPort string) *Server {
	ser.HttpPort = HttpPort
	return ser
}

func (ser *Server) SetTlsConfig(tls *tls.Config) *Server {
	ser.TlsConfig = tls
	return ser
}

//HttpServerRun run http server
func (ser *Server) HttpServerRun() {
	if len(ser.HttpPort) > 0 {
		ser.NewHTTPAPIGateway("http")
	}
}

//WsServerRun run ws server
func (ser *Server) WsServerRun() {
	if len(ser.HttpPort) > 0 {
		ser.NewHTTPAPIGateway("ws")
	}
}

//WssServerRun run wss server
func (ser *Server) WssServerRun() {
	if len(ser.HttpPort) > 0 {
		ser.NewHTTPAPIGateway("wss")
	}
}

//GrpcServerRun run grpc server
func (ser *Server) GrpcServerRun() {
	if len(ser.HttpPort) > 0 {
		ser.NewHTTPAPIGateway("grpc")
	}
}

func (ser *Server) ServerRun() {
	var address string = ser.Host + ":" + ser.Port
	listen, err := ser.MakeListener(ser.Net, address)
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
		if nil == c {
			log.Fatalln(errors.New("connect error"))
			continue
		}

		go c.DoIO()
	}
}
