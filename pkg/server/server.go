package server

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/conf"
	"github.com/HughNian/nmid/pkg/dashboard"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/metric"
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
	StartTime         time.Time
	Version           string
}

func NewServer() (ser *Server) {
	ser = &Server{
		Cpool: NewConnectPool(),
		Funcs: NewFuncMap(),
	}
	return
}

func (ser *Server) SetSConfig() *Server {
	err := conf.Init()
	if err != nil {
		return nil
	}

	SConfig := conf.GetConfig()
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

func (ser *Server) SetTlsCrtKey(certFile, keyFile string) *Server {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return ser
	}

	ser.TlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	return ser
}

func (ser *Server) SetVersion(version string) *Server {
	ser.Version = version
	return ser
}

// start up some else sever like prometheus...
func (ser *Server) SetStartUp() *Server {
	//start time
	ser.StartTime = time.Now()

	//start prometheus
	if conf.GetConfig().Prometheus.Enable {
		metric.StartServer(conf.GetConfig())

		//start dashboard
		if conf.GetConfig().Dashboard.Enable {
			dashboard := dashboard.NewDashboard(ser.StartTime, ser.Version)
			dashboard.StartDashboard(conf.GetConfig())
		}
	}

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
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				logger.Info("nmid server accept timeout, shutting down")
				break
			}

			if opErr, ok := err.(*net.OpError); ok {
				if opErr.Err.Error() == "use of closed network connection" {
					logger.Info("nmid server listener closed, shutting down")
					break
				}
			}

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
