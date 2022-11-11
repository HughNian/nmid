package server

//非tcp服务运行，http,ws,wss,grpc服务运行

import (
	"encoding/json"
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/soheilhy/cmux"
	"github.com/vmihailenco/msgpack"
	"io"
	"net"
	"net/http"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"strconv"
	"strings"
	"time"
)

var (
	ErrServerClosed  = errors.New("http: Server closed")
	ErrReqReachLimit = errors.New("request reached rate limit")
)

//NewHTTPAPIGateway gateway init
func (ser *Server) NewHTTPAPIGateway(network string) {
	if network != "http" && network != "ws" && network != "wss" && network != "grpc" {
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

	logger.Info("rpc http server start ok")

	ser.Cm = cmux.New(ln)

	httpLn := ser.Cm.Match(cmux.HTTP1Fast())
	go ser.StartHTTPAPIGateway(httpLn)

	go ser.Cm.Serve()
}

//StartHTTPAPIGateway start http api gateway
func (ser *Server) StartHTTPAPIGateway(ln net.Listener) {
	router := httprouter.New()
	router.POST("/*functionName", ser.HTTPAPIGatewayHandle)
	router.GET("/*functionName", ser.HTTPAPIGatewayHandle)
	router.PUT("/*functionName", ser.HTTPAPIGatewayHandle)

	ser.Lock()
	ser.HTTPServerGateway = &http.Server{
		Handler:      router,
		ReadTimeout:  model.DEFAULT_TIME_OUT,
		WriteTimeout: model.DEFAULT_TIME_OUT,
	}
	ser.Unlock()

	if err := ser.HTTPServerGateway.Serve(ln); err != nil {
		if err == ErrServerClosed || errors.Is(err, cmux.ErrListenerClosed) {
			logger.Error("gateway server closed")
		} else {
			logger.Errorf("error in gateway serve: %T %s", err, err)
		}
	}
}

//HTTPAPIGatewayHandle http server router handle
func (ser *Server) HTTPAPIGatewayHandle(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	requestType := r.Header.Get(model.NRequestType)

	if requestType == model.HTTPDOWORK {
		//client do work
		ser.HTTPDoWorkHandle(w, r, params)
	}
}

//HTTPDoWorkHandle http server router handle
//first get functionName
//second make nwe job
//last doWork with job
func (ser *Server) HTTPDoWorkHandle(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var err error
	var paramsType uint32
	var paramsHandleType uint32
	var paramsBytes []byte

	if r.Header.Get(model.NFunctionName) == "" {
		functionName := params.ByName("functionName")
		functionName = strings.TrimPrefix(functionName, "/")
		r.Header.Set(model.NFunctionName, functionName)
	}
	functionName := r.Header.Get(model.NFunctionName)

	paramsType = model.PARAMS_TYPE_MSGPACK
	if ptype := r.Header.Get(model.NParamsType); ptype != "" {
		val, err := strconv.Atoi(ptype)
		if nil == err {
			paramsType = uint32(val)
		}
	}
	paramsHandleType = model.PARAMS_HANDLE_TYPE_ENCODE
	if phtype := r.Header.Get(model.NParamsHandleType); phtype != "" {
		val, err := strconv.Atoi(phtype)
		if nil == err {
			paramsHandleType = uint32(val)
		}
	}

	//set headers
	wh := w.Header()
	wh.Set(model.NFunctionName, r.Header.Get(model.NFunctionName))

	//get best worker with function name
	worker := ser.Funcs.GetBestWorker(functionName)
	if worker == nil {
		wh.Set(model.NMessageStatusType, "PDT_CANT_DO")
		err = errors.New("no worker can do")
		wh.Set(model.NErrorMessage, err.Error())
		return
	}

	//make new job
	clen := r.ContentLength
	if clen == 0 {
		wh.Set(model.NMessageStatusType, "PARAMS LENGTH ZERO")
		err = errors.New("params length zero")
		wh.Set(model.NErrorMessage, err.Error())
		return
	}
	body := make([]byte, clen)
	r.Body.Read(body)
	if len(body) == 0 {
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

	job := NewJobData(functionName, string(paramsBytes))
	job.Lock()
	job.WorkerId = worker.WorkerId
	job.HTTPClientR = r
	job.FuncName = functionName
	job.Params = paramsBytes
	job.ParamsType = paramsType
	job.ParamsHandleType = paramsHandleType
	job.Unlock()

	if ok := worker.Jobs.PushJobData(job); ok {
		worker.Lock()
		worker.JobNum++
		worker.Unlock()
	} else {
		wh.Set(model.NMessageStatusType, "WORKER JOB PUSH JOBLIST ERROR")
		err = errors.New("worker job push jobList error")
		wh.Set(model.NErrorMessage, err.Error())
		return
	}

	//doWork
	go worker.doWork(job)

	//http client response
	var timer = time.After(model.DEFAULT_TIME_OUT)
	select {
	case <-worker.HttpResTag:
		{
			var retStruct model.RetStruct
			err := msgpack.Unmarshal(worker.Res.Ret, &retStruct)
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
		}
	case <-timer:
		w.WriteHeader(200)
		w.Header().Set(model.NMessageStatusType, "REQUST TIME OUT")
		wh.Set(model.NErrorMessage, model.RESTIMEOUT.Error())
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
