package server

import (
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	"github.com/soheilhy/cmux"
	"io"
	"net"
	"net/http"
	"strings"
)

const (
	NMessageStatusType = "N-NMID-MessageStatusType"
	NErrorMessage      = "N-NMID-ErrorMessage"
	NPdtDataType       = "N-NMID-PdtDataType"
	NFunctionName      = "N-NMID-FunctionName"
)

var (
	ErrServerClosed  = errors.New("http: Server closed")
	ErrReqReachLimit = errors.New("request reached rate limit")
)

//NewHTTPAPIGateway gateway init
func (ser *Server) NewHTTPAPIGateway(network string) {
	if network == "tcp" {
		logrus.Println("http gateway only use http")
		return
	}

	if len(ser.HttpPort) == 0 {
		logrus.Println("http gateway empty")
		return
	}

	address := ser.Host + ":" + ser.HttpPort
	httpLn, err := ser.MakeListener(network, address)
	if err != nil {
		logrus.Println("make http listen err", err)
		return
	}

	go ser.StartHTTPAPIGateway(httpLn)
}

//StartHTTPAPIGateway start http api gateway
func (ser *Server) StartHTTPAPIGateway(ln net.Listener) {
	router := httprouter.New()
	router.POST("/*functionName", ser.HTTPAPIGatewayHandle)
	router.GET("/*functionName", ser.HTTPAPIGatewayHandle)
	router.PUT("/*functionName", ser.HTTPAPIGatewayHandle)

	ser.Lock()
	ser.HTTPServerGateway = &http.Server{Handler: router}
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
//first get functionName
//second make nwe job
//last doWork with job
func (ser *Server) HTTPAPIGatewayHandle(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
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
	fmt.Println(`body`, string(body))
	r.Body.Read(body)

	job := NewJobData(functionName, string(body))
	job.Lock()
	job.WorkerId = worker.WorkerId
	job.HTTPClientW = w
	job.HTTPClientR = r
	job.FuncName = functionName
	job.Params = body
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
		err = errors.New("worker job push joblist error")
		wh.Set(NErrorMessage, err.Error())
		return
	}

	//doWork
	worker.doWork(job)
}

func nmidPrefixByteMatcher() cmux.Matcher {
	return func(r io.Reader) bool {
		buf := make([]byte, 1)
		n, _ := r.Read(buf)
		return n == 1 && buf[0] == 0
	}
}
