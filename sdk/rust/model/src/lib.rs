use std::time::Duration;

pub mod models;

pub const MAX_POOL_SIZE: usize = 128;
pub const QUEUE_SIZE: usize    = 1;
pub const BUFFER_SIZE: usize     = 512;
pub const MIN_DATA_SIZE: usize   = 12;
pub const UINT32_SIZE: u32     = 4;
pub const MAX_NOJOB_NUM: u32   = 10;
pub const PARAMS_SCOPE: &str   = "::";
pub const DIAL_TIME_OUT: Duration   = Duration::from_secs(5);            //连接超时
pub const DEFAULTHEARTBEATTIME: u64   = 10;
pub const NMID_SERVER_TIMEOUT: u64  = 60000; //60s nmid server 超时时间

//package data type
pub const PDT_OK: u32               = 1;
pub const PDT_ERROR: u32            = 2;
pub const PDT_CAN_DO: u32           = 3;
pub const PDT_CANT_DO: u32          = 4;
pub const PDT_NO_JOB: u32           = 5;
pub const PDT_HAVE_JOB: u32         = 6;
pub const PDT_TOSLEEP: u32          = 7;
pub const PDT_WAKEUP: u32           = 8;
pub const PDT_WAKEUPED: u32         = 9;
pub const PDT_S_GET_DATA: u32       = 10;
pub const PDT_S_RETURN_DATA: u32    = 11;
pub const PDT_S_HEARTBEAT_PONG: u32 = 23;
pub const PDT_W_GRAB_JOB: u32       = 12;
pub const PDT_W_ADD_FUNC: u32       = 13;
pub const PDT_W_DEL_FUNC: u32       = 14;
pub const PDT_W_RETURN_DATA: u32    = 15;
pub const PDT_W_HEARTBEAT_PING: u32 = 22;
pub const PDT_W_SET_NAME: u32       = 23;
pub const PDT_C_DO_JOB: u32         = 16;
pub const PDT_C_GET_DATA: u32       = 17;
pub const PDT_RATELIMIT: u32        = 18;
pub const PDT_BREAKER: u32          = 19;
pub const PDT_SC_REG_SERVICE: u32   = 19;
pub const PDT_SC_OFF_SERVICE: u32   = 20;
pub const PDT_S_REG_SERVICE_OK: u32 = 21;

pub const CONN_TYPE_INIT:u32              = 0;
pub const CONN_TYPE_SERVER:u32            = 1;
pub const CONN_TYPE_WORKER:u32            = 2;
pub const CONN_TYPE_CLIENT:u32            = 3;
pub const CONN_TYPE_SERVICE:u32           = 4;
pub const PARAMS_TYPE_MSGPACK:u32         = 5;
pub const PARAMS_TYPE_JSON:u32            = 6;
pub const JOB_STATUS_INIT:u32             = 7;
pub const JOB_STATUS_DOING :u32           = 8;
pub const JOB_STATUS_DONE:u32             = 9;
pub const PARAMS_HANDLE_TYPE_ENCODE:u32   = 10;
pub const PARAMS_HANDLE_TYPE_ORIGINAL:u32 = 11;