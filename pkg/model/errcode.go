package model

import "github.com/HughNian/github.com/HughNian/nmid/pkg/errno"

var (
	//RoteError -100 ~ -300
	RoteError              = &errno.Errno{-100, "Request format Error"}           // 解析请求route错误
	TokenParseError        = &errno.Errno{-101, "Request token invalid"}          // 解析token错误
	RelationError          = &errno.Errno{-102, "The relation is invalid"}        // 调用链不成立
	ParamError             = &errno.Errno{-104, "The required parameter is null"} // 目标服务的信息不存在
	RequestError           = &errno.Errno{-105, "Request Error"}                  // 请求错误
	ResponseError          = &errno.Errno{-106, "Response Error"}                 // 响应错误
	NewLimiterError        = &errno.Errno{-107, "init limiter Error"}             // 初始化限流错误
	RemoteHostError        = &errno.Errno{-108, "remote host Error"}              // 目标服务的host错误
	FuseError              = &errno.Errno{-499, "The fuse Error"}                 // 熔断错误
	RequestBodyLimitError  = &errno.Errno{-413, "Request Error: body size exceeds the given limit"}
	ResponseBodyLimitError = &errno.Errno{-500, "Response Error: body size exceeds the given limit"}
	JsonMarshalError       = &errno.Errno{-202, "Json marshal Error"}             // json数据错误
	JsonUnmarshalError     = &errno.Errno{-203, "Json unmarshal Error"}           // json解析数据错误
	ConfigError            = &errno.Errno{-204, "Config Error"}                   // 配置错误
	TargetResponseWaring   = &errno.Errno{-300, "target service response waring"} //目标服务返回异常告警
)
