package server

import (
	"crypto/tls"
	"errors"
	"github.com/soheilhy/cmux"
	"log"
	"net"
	"net/http"
	"nmid-v2/pkg/model"
	"sync"
)

//rpc server, can use tcp、http、ws etc

type Server struct {
	sync.Mutex

	Host              string
	Port              string
	HttpPort          string
	Net               string
	Ln                net.Listener
	Cm                cmux.CMux
	HTTPServerGateway *http.Server
	SConfig           model.ServerConfig
	Cpool             *ConnectPool
	Funcs             *FuncMap
	TlsConfig         *tls.Config
}

func NewServer() (ser *Server) {
	ser = &Server{
		Cpool: NewConnectPool(),
		Funcs: NewFuncMap(),
	}
	return
}

func (ser *Server) SetSConfig(SConfig model.ServerConfig) *Server {
	ser.SConfig = SConfig
	ser.Net = SConfig.Server.NETWORK
	ser.Host = SConfig.Server.HOST
	ser.Port = SConfig.Server.PORT
	ser.HttpPort = SConfig.Server.HTTPPORT
	return ser
}

func (ser *Server) SetNet(net string) *Server {
	ser.Net = net
	return ser
}

func (ser *Server) SetHost(host string) *Server {
	ser.Host = host
	return ser
}

func (ser *Server) SetPort(port string) *Server {
	ser.Port = port
	return ser
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
	listen, err := ser.NewListener(ser.Net, address)
	if err != nil {
		log.Fatalln(err)
		panic(err)
	}
	ser.Ln = listen

	for {
		conn, err := ser.Ln.Accept()
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

func (ser *Server) ServerClose(wg *sync.WaitGroup) {
	defer wg.Done()

	ser.Cm.Close()
	ser.HTTPServerGateway.Close()
	ser.Ln.Close()
}
