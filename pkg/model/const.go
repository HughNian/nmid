package model

import (
	"errors"
	"time"
)

const (
	MAX_POOL_SIZE        = 128
	QUEUE_SIZE           = 1
	BUFFER_SIZE          = 512
	MIN_DATA_SIZE        = 12
	UINT32_SIZE          = 4
	MAX_NOJOB_NUM        = 10
	PARAMS_SCOPE         = "::"
	DEFAULT_TIME_OUT     = 30 * time.Millisecond             //io超时
	DIAL_TIME_OUT        = 6 * time.Second                   //连接超时
	CLIENT_ALIVE_TIME    = time.Duration(1800) * time.Second //客户端长连接生存周期
	DEFAULTHEARTBEATTIME = 10 * time.Second
	NMID_SERVER_TIMEOUT  = 20000 //20s //nmid server 超时时间

	//package data type
	PDT_OK               = 1
	PDT_ERROR            = 2
	PDT_CAN_DO           = 3
	PDT_CANT_DO          = 4
	PDT_NO_JOB           = 5
	PDT_HAVE_JOB         = 6
	PDT_TOSLEEP          = 7
	PDT_WAKEUP           = 8
	PDT_WAKEUPED         = 9
	PDT_S_GET_DATA       = 10
	PDT_S_RETURN_DATA    = 11
	PDT_S_HEARTBEAT_PONG = 23
	PDT_W_GRAB_JOB       = 12
	PDT_W_ADD_FUNC       = 13
	PDT_W_DEL_FUNC       = 14
	PDT_W_RETURN_DATA    = 15
	PDT_W_HEARTBEAT_PING = 22
	PDT_C_DO_JOB         = 16
	PDT_C_GET_DATA       = 17
	PDT_RATELIMIT        = 18
	PDT_BREAKER          = 19
	PDT_SC_REG_SERVICE   = 19
	PDT_SC_OFF_SERVICE   = 20
	PDT_S_REG_SERVICE_OK = 21
)

// connect types & status
const (
	CONN_TYPE_INIT              = 0
	CONN_TYPE_SERVER            = 1
	CONN_TYPE_WORKER            = 2
	CONN_TYPE_CLIENT            = 3
	CONN_TYPE_SERVICE           = 4
	PARAMS_TYPE_MSGPACK         = 5
	PARAMS_TYPE_JSON            = 6
	JOB_STATUS_INIT             = 7
	JOB_STATUS_DOING            = 8
	JOB_STATUS_DONE             = 9
	PARAMS_HANDLE_TYPE_ENCODE   = 10
	PARAMS_HANDLE_TYPE_ORIGINAL = 11
)

// loadblance type
const (
	LOADBLANCE_HASH         = 1
	LOADBLANCE_LRU          = 2
	LOADBLANCE_ROUND_WEIGHT = 3
)

// ratelimit
const (
	MAXPENDING     = 1000
	MAXOUTSTANDING = 10
	TARGETLATENCY  = 5 * time.Millisecond
	FILLINTERVAL   = 1 * time.Second
	CAPACITY       = 1000
)

// http request
const (
	NRequestType       = "N-NMID-RequestType"
	NParamsType        = "N-NMID-ParamsType"
	NParamsHandleType  = "N-NMID-ParamsHandleType"
	NMessageStatusType = "N-NMID-MessageStatusType"
	NErrorMessage      = "N-NMID-ErrorMessage"
	NPdtDataType       = "N-NMID-PdtDataType"
	NFunctionName      = "N-NMID-FunctionName"
)

var (
	RESTIMEOUT               = errors.New("RESTIMEOUT")
	HTTPDOWORK               = "HTTP_DO_WORK"
	HTTPADDSERVICE           = "HTTP_ADD_SERVICE"
	PARAMSTYPEMSGPACK        = "5"
	PARAMSTYPEJSON           = "6"
	PARAMSHANDLETYPEENCODE   = "10"
	PARAMSHANDLETYPEORIGINAL = "11"
)
