package model

import "github.com/HughNian/nmid/pkg/errno"

var (
	//RoteError -100 ~ -300
	RoteError              = &errno.Errno{State: -100, Msg: "Request format Error"}           // 解析请求route错误
	TokenParseError        = &errno.Errno{State: -101, Msg: "Request token invalid"}          // 解析token错误
	RelationError          = &errno.Errno{State: -102, Msg: "The relation is invalid"}        // 调用链不成立
	ParamError             = &errno.Errno{State: -104, Msg: "The required parameter is null"} // 目标服务的信息不存在
	RequestError           = &errno.Errno{State: -105, Msg: "Request Error"}                  // 请求错误
	ResponseError          = &errno.Errno{State: -106, Msg: "Response Error"}                 // 响应错误
	NewLimiterError        = &errno.Errno{State: -107, Msg: "init limiter Error"}             // 初始化限流错误
	RemoteHostError        = &errno.Errno{State: -108, Msg: "remote host Error"}              // 目标服务的host错误
	FuseError              = &errno.Errno{State: -499, Msg: "The fuse Error"}                 // 熔断错误
	RequestBodyLimitError  = &errno.Errno{State: -413, Msg: "Request Error: body size exceeds the given limit"}
	ResponseBodyLimitError = &errno.Errno{State: -500, Msg: "Response Error: body size exceeds the given limit"}
	JsonMarshalError       = &errno.Errno{State: -202, Msg: "Json marshal Error"}             // json数据错误
	JsonUnmarshalError     = &errno.Errno{State: -203, Msg: "Json unmarshal Error"}           // json解析数据错误
	ConfigError            = &errno.Errno{State: -204, Msg: "Config Error"}                   // 配置错误
	TargetResponseWaring   = &errno.Errno{State: -300, Msg: "target service response waring"} //目标服务返回异常告警
)
