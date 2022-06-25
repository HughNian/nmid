package server

import "time"

const (
	MAX_POOL_SIZE     = 128
	QUEUE_SIZE        = 1
	BUFFER_SIZE       = 512
	MIN_DATA_SIZE     = 12
	UINT32_SIZE       = 4
	MAX_NOJOB_NUM     = 10
	PARAMS_SCOPE      = 0x3A
	CLIENT_ALIVE_TIME = 30 * 60 * time.Second //客户端长连接生存周期

	//package data type
	PDT_OK            = 1
	PDT_ERROR         = 2
	PDT_CAN_DO        = 3
	PDT_CANT_DO       = 4
	PDT_NO_JOB        = 5
	PDT_HAVE_JOB      = 6
	PDT_TOSLEEP       = 7
	PDT_WAKEUP        = 8
	PDT_WAKEUPED      = 9
	PDT_S_GET_DATA    = 10
	PDT_S_RETURN_DATA = 11
	PDT_W_GRAB_JOB    = 12
	PDT_W_ADD_FUNC    = 13
	PDT_W_DEL_FUNC    = 14
	PDT_W_RETURN_DATA = 15
	PDT_C_DO_JOB      = 16
	PDT_C_GET_DATA    = 17
	PDT_RATELIMIT     = 18
	PDT_BREAKER       = 19
)

//connect status
const (
	CONN_TYPE_INIT   = 0
	CONN_TYPE_SERVER = 1
	CONN_TYPE_WORKER = 2
	CONN_TYPE_CLIENT = 3
	PARAMS_TYPE_ONE  = 4
	PARAMS_TYPE_MUL  = 5
	JOB_STATUS_INIT  = 6
	JOB_STATUS_DOING = 7
	JOB_STATUS_DONE  = 8
)

//ratelimit
const (
	MAXPENDING     = 1000
	MAXOUTSTANDING = 10
	TARGETLATENCY  = 5 * time.Millisecond
	FILLINTERVAL   = 1 * time.Second
	CAPACITY       = 1000
)
