package server

//非tcp服务运行，http,ws,wss,grpc服务运行

import (
	"encoding/json"
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"github.com/vmihailenco/msgpack"
	"io"
	"net"
	"net/http"
	cli "nmid-v2/pkg/client"
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
		logrus.Println("protocol not supported")
		return
	}

	if len(ser.HttpPort) == 0 {
		logrus.Println("http gateway empty")
		return
	}

	address := ser.Host + ":" + ser.HttpPort
	ln, err := ser.MakeListener(network, address)
	if err != nil {
		logrus.Println("make http listen err", err)
		return
	}

	m := cmux.New(ln)

	httpLn := m.Match(cmux.HTTP1Fast())
	go ser.StartHTTPAPIGateway(httpLn)

	go m.Serve()
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
		ReadTimeout:  DEFAULT_TIME_OUT,
		WriteTimeout: DEFAULT_TIME_OUT,
	}
	ser.Unlock()

	if err := ser.HTTPServerGateway.Serve(ln); err != nil {
		if err == ErrServerClosed || errors.Is(err, cmux.ErrListenerClosed) {
			logrus.Println("gateway server closed")
		} else {
			logrus.Println("error in gateway Serve: %T %s", err, err)
		}
	}
}

//HTTPAPIGatewayHandle http server router handle
func (ser *Server) HTTPAPIGatewayHandle(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	requestType := r.Header.Get(NRequestType)

	if requestType == HTTPDOWORK {
		//client do work
		ser.HTTPDoWorkHandle(w, r, params)
	} else if requestType == HTTPADDSERVICE {
		//service add service
		ser.HTTPAddServiceHandle(w, r, params)
	}
}

//HTTPDoWorkHandle http server router handle
//first get functionName
//second make nwe job
//last doWork with job
func (ser *Server) HTTPDoWorkHandle(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var err error
	if r.Header.Get(NFunctionName) == "" {
		functionName := params.ByName("functionName")
		functionName = strings.TrimPrefix(functionName, "/")
		r.Header.Set(NFunctionName, functionName)
	}
	functionName := r.Header.Get(NFunctionName)

	//set headers
	wh := w.Header()
	wh.Set(NFunctionName, r.Header.Get(NFunctionName))

	//get best worker with function name
	worker := ser.Funcs.GetBestWorker(functionName)
	if worker == nil {
		wh.Set(NMessageStatusType, "PDT_CANT_DO")
		err = errors.New("no worker can do")
		wh.Set(NErrorMessage, err.Error())
		return
	}

	//make new job
	clen := r.ContentLength
	if clen == 0 {
		wh.Set(NMessageStatusType, "PARAMS LENGTH ZERO")
		err = errors.New("params length zero")
		wh.Set(NErrorMessage, err.Error())
		return
	}
	body := make([]byte, clen)
	r.Body.Read(body)
	if len(body) == 0 {
		wh.Set(NMessageStatusType, "READ PARAMS EMPTY")
		err = errors.New("read params empty")
		wh.Set(NErrorMessage, err.Error())
		return
	}

	var paramsval []string
	err = json.Unmarshal(body, &paramsval)
	if err != nil {
		wh.Set(NMessageStatusType, "JSON UNMARSHAL ERROR")
		err = errors.New("json unmarshal error")
		wh.Set(NErrorMessage, err.Error())
		return
	}
	msgpparams, err := msgpack.Marshal(&paramsval)
	if err != nil {
		wh.Set(NMessageStatusType, "PARAMS MSGPACK ERROR")
		err = errors.New("params msgpack error")
		wh.Set(NErrorMessage, err.Error())
		return
	}

	job := NewJobData(functionName, string(msgpparams))
	job.Lock()
	job.WorkerId = worker.WorkerId
	job.HTTPClientR = r
	job.FuncName = functionName
	job.Params = msgpparams
	if IsMulParams(job.Params) {
		job.ParamsType = PARAMS_TYPE_MUL
	} else {
		job.ParamsType = PARAMS_TYPE_ONE
	}
	job.Unlock()
	if ok := worker.Jobs.PushJobData(job); ok {
		worker.Lock()
		worker.JobNum++
		worker.Unlock()
	} else {
		wh.Set(NMessageStatusType, "WORKER JOB PUSH JOBLIST ERROR")
		err = errors.New("worker job push jobList error")
		wh.Set(NErrorMessage, err.Error())
		return
	}

	//doWork
	go worker.doWork(job)

	//http client response
	var timer = time.After(DEFAULT_TIME_OUT)
	select {
	case <-worker.HttpResTag:
		{
			var retStruct cli.RetStruct
			err := msgpack.Unmarshal(worker.Res.Ret, &retStruct)
			if nil != err {
				w.Header().Set(NMessageStatusType, "RET MSGPACK UNMARSHAL ERROR")
				wh.Set(NErrorMessage, err.Error())
				return
			}

			if retStruct.Code != 0 {
				w.Header().Set(NMessageStatusType, "RET CODE ERROR")
				wh.Set(NErrorMessage, retStruct.Msg)
				return
			}

			w.Header().Set(NPdtDataType, "PDT_S_RETURN_DATA")
			w.Write(retStruct.Data)
		}
	case <-timer:
		w.WriteHeader(200)
		w.Header().Set(NMessageStatusType, "REQUST TIME OUT")
		wh.Set(NErrorMessage, RESTIMEOUT.Error())
		return
	}
}

func (ser *Server) HTTPAddServiceHandle(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

}

func nmidPrefixByteMatcher() cmux.Matcher {
	return func(r io.Reader) bool {
		buf := make([]byte, 1)
		n, _ := r.Read(buf)
		return n == 1 && buf[0] == 0
	}
}
