package worker

import (
	"context"
	"encoding/binary"
	"fmt"
	"runtime"
	"strconv"
	"time"

	cli "github.com/HughNian/nmid/pkg/client"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/trace"
	"github.com/HughNian/nmid/pkg/utils"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/propagation"
	"github.com/vmihailenco/msgpack"
	v3 "skywalking.apache.org/repo/goapi/collect/language/agent/v3"
)

type Response struct {
	DataType uint32
	Data     []byte
	DataLen  uint32

	Handle           string
	HandleLen        uint32
	ParamsType       uint32
	ParamsHandleType uint32
	ParamsLen        uint32
	Params           []byte
	ParamsMap        map[string]interface{}
	JobId            string
	JobIdLen         uint32
	Ret              []byte
	RetLen           uint32
	Sw8              string
	Sw8Len           uint32

	Agent         *Agent
	TraceEntryCtx context.Context
}

func NewRes() (res *Response) {
	res = &Response{
		Data:       make([]byte, 0),
		ParamsType: model.PARAMS_TYPE_MSGPACK,
		Ret:        make([]byte, 0),
	}
	return
}

// DecodePack 解包
func DecodePack(data []byte) (resp *Response, resLen int, err error) {
	resLen = len(data)
	if resLen < model.MIN_DATA_SIZE {
		err = fmt.Errorf("invalid data: %v", data)
		return
	}
	cl := int(binary.BigEndian.Uint32(data[8:model.MIN_DATA_SIZE]))
	if resLen < model.MIN_DATA_SIZE+cl {
		err = fmt.Errorf("invalid data: %v", data)
		return
	}
	content := data[model.MIN_DATA_SIZE : model.MIN_DATA_SIZE+cl]
	if len(content) != cl {
		err = fmt.Errorf("invalid data: %v", data)
		return
	}

	resp = NewRes()
	resp.DataType = binary.BigEndian.Uint32(data[4:8])
	resp.DataLen = uint32(cl)
	resp.Data = content

	if resp.DataType == model.PDT_S_GET_DATA {
		//新的解包协议
		start := model.MIN_DATA_SIZE
		end := model.MIN_DATA_SIZE + model.UINT32_SIZE
		resp.ParamsType = uint32(binary.BigEndian.Uint32(data[start:end]))
		start = end
		end = start + model.UINT32_SIZE
		resp.ParamsHandleType = uint32(binary.BigEndian.Uint32(data[start:end]))
		start = end
		end = start + model.UINT32_SIZE
		resp.HandleLen = binary.BigEndian.Uint32(data[start:end])
		start = end
		end = start + model.UINT32_SIZE
		resp.ParamsLen = binary.BigEndian.Uint32(data[start:end])
		start = end
		end = start + model.UINT32_SIZE
		resp.JobIdLen = binary.BigEndian.Uint32(data[start:end])
		start = end
		end = start + int(resp.HandleLen)
		resp.Handle = string(data[start:end])
		start = end
		end = start + int(resp.ParamsLen)
		resp.ParseParams(data[start:end])
		start = end
		end = start + int(resp.JobIdLen)
		resp.JobId = string(data[start:end])
	}

	return
}

func (resp *Response) GetResponse() *Response {
	return resp
}

func (resp *Response) ParseParams(params []byte) {
	resp.Params = params
	if resp.ParamsType == model.PARAMS_TYPE_MSGPACK {
		resp.ParamsMap = utils.MsgpackParamsMap(params)
	} else if resp.ParamsType == model.PARAMS_TYPE_JSON {
		resp.ParamsMap = utils.JsonParamsMap(params)
	}
}

func (resp *Response) GetParams() []byte {
	if resp.ParamsLen == 0 {
		return nil
	}

	return resp.Params
}

func (resp *Response) GetParamsMap() map[string]interface{} {
	if resp.ParamsLen == 0 {
		return nil
	}

	return resp.ParamsMap
}

// SetEntrySpan do entry span 入口span
func (resp *Response) SetEntrySpan() {
	if !resp.Agent.Worker.useTrace {
		return
	}

	workerId := resp.Agent.Worker.WorkerId
	workerName := resp.Agent.Worker.WorkerName

	tracer := resp.Agent.Worker.Tracer
	if nil == tracer {
		return
	}

	var sw8 string
	span, entryCtx, err := tracer.CreateEntrySpan(context.TODO(), resp.Handle, func(key string) (string, error) {
		if val, exist := resp.ParamsMap[key]; exist {
			sw8 = val.(string)
		}
		return sw8, nil
	})
	if err != nil {
		logger.Warnf("inflow trace CreateEntrySpan error sw8:"+sw8, err)
		return
	}

	if sw8 == "" {
		s := span.(go2sky.ReportedSpan)
		sw8 = (&propagation.SpanContext{
			TraceID:               s.Context().TraceID,
			ParentSegmentID:       s.Context().ParentSegmentID,
			ParentService:         workerName,
			ParentServiceInstance: workerId,
			ParentEndpoint:        resp.Handle,
			ParentSpanID:          -1, // 首层固定
			Sample:                1,
		}).EncodeSW8()
	}

	resp.TraceEntryCtx = entryCtx
	resp.Sw8Len = uint32(len(sw8))
	resp.Sw8 = sw8

	span.SetSpanLayer(v3.SpanLayer_RPCFramework)
	span.SetOperationName(resp.Handle)
	span.SetComponent(trace.ComponentIDGoMicroServer)
	span.Tag("go_version", runtime.Version())
	span.Tag(go2sky.TagStatusCode, strconv.Itoa(200))
	span.Log(time.Now(), "[nmid rpc]", fmt.Sprintf("start request, workerid:%s, workername:%s, function:%s", workerId, workerName, resp.Handle))
	span.End()
}

// SetExitSpan do exit span 出口span
func (resp *Response) SetExitSpan(serverAddr, exitHandle string, params *map[string]interface{}) {
	if !resp.Agent.Worker.useTrace {
		return
	}

	tracer := resp.Agent.Worker.Tracer
	if nil == tracer {
		return
	}

	operationName := serverAddr + "@" + exitHandle
	span, err := tracer.CreateExitSpan(resp.TraceEntryCtx, operationName, serverAddr, func(key, value string) error {
		//(*params)[propagation.Header] = resp.Sw8
		(*params)[key] = value
		return nil
	})
	if err != nil {
		return
	}

	span.SetOperationName(operationName)
	span.SetComponent(trace.ComponentIDGoMicroClient)
	span.Tag(go2sky.TagURL, serverAddr)
	span.SetSpanLayer(v3.SpanLayer_RPCFramework)
	span.End()
}

// ClientCall call next worker
func (resp *Response) ClientCall(serverAddr, funcName string, params map[string]interface{}, respHandler func(resp *cli.Response), errHandler func(e error)) {
	if len(serverAddr) == 0 {
		logger.Fatal("serverAddr must be have")
	}

	if len(funcName) == 0 {
		logger.Fatal("funcName must be have")
	}

	var client *cli.Client
	var err error

	client, err = cli.NewClient("tcp", serverAddr).Start()
	if nil == client || err != nil {
		logger.Error(err)
	}
	defer client.Close()

	client.ErrHandler = errHandler

	//use trace
	if resp.Agent.Worker.useTrace {
		//set skywalking exit span
		resp.SetExitSpan(serverAddr, funcName, &params)
	}

	//请求下层worker
	cparams, err := msgpack.Marshal(&params)
	if err != nil {
		logger.Error("params msgpack error:", err)
	}
	err = client.Do(funcName, cparams, respHandler)
	if nil != err {
		logger.Error(`--do err--`, err)
	}
}

func discovery(funcName string) *cli.Client {
	var consumer *cli.Consumer

	client := consumer.Discovery(funcName)
	if client != nil {
		client, err := client.SetIoTimeOut(30 * time.Second).Start()
		if nil == client || err != nil {
			logger.Error(err)
		}
	}

	return client
}

// ClientCall call next worker
func (resp *Response) ClientDiscovery(funcName string, params map[string]interface{}, respHandler func(resp *cli.Response), errHandler func(e error)) {
	if len(funcName) == 0 {
		logger.Fatal("funcName must be have")
	}

	var client *cli.Client
	var err error

	client = discovery(funcName)
	if nil == client {
		logger.Fatal("funcName not found")
	}
	defer client.Close()

	client.ErrHandler = errHandler

	//use trace
	if resp.Agent.Worker.useTrace {
		//set skywalking exit span
		resp.SetExitSpan(client.Addr, funcName, &params)
	}

	//请求下层worker
	cparams, err := msgpack.Marshal(&params)
	if err != nil {
		logger.Error("params msgpack error:", err)
	}
	err = client.Do(funcName, cparams, respHandler)
	if nil != err {
		logger.Error(`--do err--`, err)
	}
}
