package server

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"

	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/soheilhy/cmux"
)

//rpc server

type Server struct {
	sync.Mutex

	Host              string
	Port              string
	HttpPort          string
	WSPort            string
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
	ser.Net = SConfig.RpcServer.NETWORK
	ser.Host = SConfig.RpcServer.HOST
	ser.Port = SConfig.RpcServer.PORT
	ser.HttpPort = SConfig.RpcServer.HTTPPORT
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

// HttpServerRun run http server
func (ser *Server) HttpServerRun() {
	if len(ser.HttpPort) > 0 {
		ser.NewHTTPAPIGateway("http")
	}
}

// WsServerRun run ws server
func (ser *Server) WsServerRun() {
	if len(ser.HttpPort) > 0 {
		ser.NewHTTPAPIGateway("ws")
	}
}

// WssServerRun run wss server
func (ser *Server) WssServerRun() {
	if len(ser.HttpPort) > 0 {
		ser.NewHTTPAPIGateway("wss")
	}
}

// GrpcServerRun run grpc server
func (ser *Server) GrpcServerRun() {
	if len(ser.HttpPort) > 0 {
		ser.NewHTTPAPIGateway("grpc")
	}
}

func (ser *Server) ServerRun() {
	var address string = ser.Host + ":" + ser.Port
	listen, err := ser.NewListener(ser.Net, address)
	if err != nil {
		logger.Fatalf("listener err %s", err.Error())
	}
	ser.Ln = listen

	logger.Info("rpc tcp server start ok at port: ", ser.Port)

	for {
		conn, err := ser.Ln.Accept()
		if err != nil {
			logger.Errorf("accept err %s", err.Error())
			continue
		}

		c := ser.Cpool.NewConnect(ser, conn)
		if nil == c {
			logger.Errorf("connect err %s", errors.New("connect error or forbidden"))
			continue
		}

		go c.DoIO()
	}
}

func (ser *Server) ServerClose(wg *sync.WaitGroup) {
	defer wg.Done()

	ser.Ln.Close()

	if ser.HTTPServerGateway != nil {
		ser.Cm.Close()
		ser.HTTPServerGateway.Close()
	}
}
