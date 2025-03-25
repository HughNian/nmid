package server

//非tcp服务运行，http,ws,wss,grpc服务运行

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"

	cli "github.com/HughNian/nmid/pkg/client"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/julienschmidt/httprouter"
	"github.com/soheilhy/cmux"
	"github.com/vmihailenco/msgpack"
	"golang.org/x/net/websocket"
)

var (
	ErrServerClosed  = errors.New("http: Server closed")
	ErrReqReachLimit = errors.New("request reached rate limit")
)

// NewHTTPAPIGateway gateway init
func (ser *Server) NewHTTPAPIGateway(network string) {
	if network != "http" && network != "grpc" {
		logger.Error("protocol not supported")
		return
	}

	if len(ser.HttpPort) == 0 {
		logger.Error("http gateway empty")
		return
	}

	address := ser.Host + ":" + ser.HttpPort
	ln, err := ser.NewListener(network, address)
	if err != nil {
		logger.Error("make http listen err", err)
		return
	}

	logger.Info("rpc http server start ok at port: ", ser.HttpPort)

	ser.Cm = cmux.New(ln)

	httpLn := ser.Cm.Match(cmux.HTTP1Fast())
	go ser.StartHTTPAPIGateway(httpLn)

	go ser.Cm.Serve()
}

// StartHTTPAPIGateway start http api gateway
func (ser *Server) StartHTTPAPIGateway(ln net.Listener) {
	router := httprouter.New()
	router.POST("/*functionName", ser.HTTPAPIGatewayHandle)
	router.GET("/*functionName", ser.HTTPAPIGatewayHandle)
	router.PUT("/*functionName", ser.HTTPAPIGatewayHandle)

	ser.Lock()
	ser.HTTPServerGateway = &http.Server{
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second,
	}
	ser.Unlock()

	if err := ser.HTTPServerGateway.Serve(ln); err != nil {
		if err == ErrServerClosed || errors.Is(err, cmux.ErrListenerClosed) {
			logger.Error("gateway server closed")
		} else {
			logger.Error("error in gateway serve:", err)
		}
	}
}

// HTTPAPIGatewayHandle http server router handle
func (ser *Server) HTTPAPIGatewayHandle(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	requestType := r.Header.Get(model.NRequestType)

	if requestType == model.HTTPDOWORK {
		//client do work
		ser.HTTPDoWorkByClientHandle(w, r, params)
	}
}

func (ser *Server) GetClient() *cli.Client {
	serverAddr := fmt.Sprintf("%s:%s", ser.Host, ser.Port)
	client, err := cli.NewClient("tcp", serverAddr).SetIoTimeOut(30 * time.Second).Start()
	if nil == client || err != nil {
		logger.Error(err)
	}

	return client
}

func (ser *Server) HTTPDoWorkByClientHandle(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var functionName string
	var paramsType uint32
	var paramsBytes []byte
	var err error

	functionName = r.Header.Get(model.NFunctionName)
	if r.Header.Get(model.NFunctionName) == "" {
		functionName = params.ByName("functionName")
	}

	paramsType = model.PARAMS_TYPE_MSGPACK
	if ptype := r.Header.Get(model.NParamsType); ptype != "" {
		val, err := strconv.Atoi(ptype)
		if nil == err {
			paramsType = uint32(val)
		}
	}

	client := ser.GetClient()
	defer client.Close()
	client.SetParamsType(paramsType)

	//set headers
	wh := w.Header()
	wh.Set(model.NFunctionName, r.Header.Get(model.NFunctionName))

	client.ErrHandler = func(e error) {
		if model.RESTIMEOUT == e {
			logger.Warn("time out here")
			wh.Set(model.NMessageStatusType, "REQUST TIME OUT")
			wh.Set(model.NErrorMessage, e.Error())
			w.WriteHeader(408)
		}

		logger.Error(e)
		wh.Set(model.NMessageStatusType, "REQUST ERROR")
		wh.Set(model.NErrorMessage, e.Error())
		w.WriteHeader(500)
	}

	respHandler := func(resp *cli.Response) {
		if resp.DataType == model.PDT_S_RETURN_DATA && resp.RetLen != 0 {
			if resp.RetLen == 0 {
				logger.Info("ret empty")
				return
			}

			var retStruct model.RetStruct
			err := msgpack.Unmarshal(resp.Ret, &retStruct)
			if nil != err {
				w.Header().Set(model.NMessageStatusType, "RET MSGPACK UNMARSHAL ERROR")
				wh.Set(model.NErrorMessage, err.Error())
				return
			}

			if retStruct.Code != 0 {
				w.Header().Set(model.NMessageStatusType, "RET CODE ERROR")
				wh.Set(model.NErrorMessage, retStruct.Msg)
				return
			}

			w.Header().Set(model.NPdtDataType, "PDT_S_RETURN_DATA")
			w.Write(retStruct.Data)
			return
		}
	}

	clen := r.ContentLength
	if clen == 0 {
		wh.Set(model.NMessageStatusType, "PARAMS LENGTH ZERO")
		err = errors.New("params length zero")
		wh.Set(model.NErrorMessage, err.Error())
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		wh.Set(model.NMessageStatusType, "READ PARAMS EMPTY")
		err = errors.New("read params empty")
		wh.Set(model.NErrorMessage, err.Error())
		return
	}

	if paramsType == model.PARAMS_TYPE_MSGPACK {
		paramsval := make(map[string]interface{})
		err = json.Unmarshal(body, &paramsval)
		if err != nil {
			wh.Set(model.NMessageStatusType, "JSON UNMARSHAL ERROR")
			err = errors.New("json unmarshal error")
			wh.Set(model.NErrorMessage, err.Error())
			return
		}

		paramsBytes, err = msgpack.Marshal(&paramsval)
		if err != nil {
			wh.Set(model.NMessageStatusType, "PARAMS MSGPACK ERROR")
			err = errors.New("params msgpack error")
			wh.Set(model.NErrorMessage, err.Error())
			return
		}
	} else if paramsType == model.PARAMS_TYPE_JSON {
		paramsBytes = body
	}

	err = client.Do(functionName, paramsBytes, respHandler)
	if nil != err {
		wh.Set(model.NMessageStatusType, "CLIENT DO ERROR")
		err = errors.New("client do error")
		wh.Set(model.NErrorMessage, err.Error())
		return
	}
}

func nmidPrefixByteMatcher() cmux.Matcher {
	return func(r io.Reader) bool {
		buf := make([]byte, 1)
		n, _ := r.Read(buf)
		return n == 1 && buf[0] == 0
	}
}

// NewWSAPIGateway gateway init
func (ser *Server) NewWSAPIGateway(network string) {
	if network != "ws" && network != "wss" && network != "grpc" {
		logger.Error("protocol not supported")
		return
	}

	if len(ser.WSPort) == 0 {
		logger.Error("ws port empty")
		return
	}

	wsPath := "/nmidws"

	address := ser.Host + ":" + ser.WSPort
	ln, err := ser.NewListener(network, address)
	if err != nil {
		logger.Error("make ws listen err", err)
		return
	}

	logger.Info("rpc ws server start ok at port: ", ser.WSPort)

	mux := http.NewServeMux()
	mux.Handle(wsPath, websocket.Handler(ser.WSDoWorkHandle))
	srv := &http.Server{Handler: mux}

	go srv.Serve(ln)
}

func (s *Server) WSDoWorkHandle(conn *websocket.Conn) {
	//todo ws's read and write
	// s.serveConn(conn)
}
