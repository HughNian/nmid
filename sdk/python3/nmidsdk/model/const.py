MAX_POOL_SIZE        = 128
QUEUE_SIZE           = 1
BUFFER_SIZE          = 512
MIN_DATA_SIZE        = 12
UINT32_SIZE          = 4
MAX_NOJOB_NUM        = 10
PARAMS_SCOPE         = "::"
DIAL_TIME_OUT        = 6                   #连接超时
DEFAULTHEARTBEATTIME = 10
NMID_SERVER_TIMEOUT  = 60000 #60s nmid server 超时时间

#package data type
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
PDT_W_SET_NAME       = 23
PDT_C_DO_JOB         = 16
PDT_C_GET_DATA       = 17
PDT_RATELIMIT        = 18
PDT_BREAKER          = 19
PDT_SC_REG_SERVICE   = 19
PDT_SC_OFF_SERVICE   = 20
PDT_S_REG_SERVICE_OK = 21

#connect types & status
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

RESTIMEOUT               = "RESTIMEOUT"
HTTPDOWORK               = "HTTP_DO_WORK"
HTTPADDSERVICE           = "HTTP_ADD_SERVICE"
PARAMSTYPEMSGPACK        = "5"
PARAMSTYPEJSON           = "6"
PARAMSHANDLETYPEENCODE   = "10"
PARAMSHANDLETYPEORIGINAL = "11"

EtcdBaseKey = "nmid/"