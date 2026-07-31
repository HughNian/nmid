package com.nmidsdk.model;

/**
 * Constants for the nmid wire protocol.
 * Mirrors sdk/node/lib/const.js and pkg/model/const.go.
 */
public final class Constants {
    private Constants() {
        throw new AssertionError("No instances for you!");
    }

    public static final String VERSION = "v1.0.16";

    // sizes & timing (milliseconds, except where noted)
    public static final int MAX_POOL_SIZE = 128;
    public static final int QUEUE_SIZE = 1;
    public static final int BUFFER_SIZE = 512;
    public static final int MIN_DATA_SIZE = 12; // packet header: ConnType(4)+DataType(4)+DataLen(4)
    public static final int UINT32_SIZE = 4;
    public static final int MAX_NOJOB_NUM = 10;
    public static final String PARAMS_SCOPE = "::";
    public static final int DEFAULT_TIME_OUT = 30000;
    public static final int DIAL_TIME_OUT = 6000;        // ms, connect timeout
    public static final int CLIENT_ALIVE_TIME = 1800000;
    public static final int DEFAULT_HEARTBEAT_TIME = 10000; // ms, heartbeat interval
    public static final int NMID_SERVER_TIMEOUT = 60000;  // ms, server timeout before reconnect

    // package data type
    public static final int PDT_OK = 1;
    public static final int PDT_ERROR = 2;
    public static final int PDT_CAN_DO = 3;
    public static final int PDT_CANT_DO = 4;
    public static final int PDT_NO_JOB = 5;
    public static final int PDT_HAVE_JOB = 6;
    public static final int PDT_TOSLEEP = 7;
    public static final int PDT_WAKEUP = 8;
    public static final int PDT_WAKEUPED = 9;
    public static final int PDT_S_GET_DATA = 10;
    public static final int PDT_S_RETURN_DATA = 11;
    public static final int PDT_S_HEARTBEAT_PONG = 23;
    public static final int PDT_W_GRAB_JOB = 12;
    public static final int PDT_W_ADD_FUNC = 13;
    public static final int PDT_W_DEL_FUNC = 14;
    public static final int PDT_W_RETURN_DATA = 15;
    public static final int PDT_W_HEARTBEAT_PING = 22;
    public static final int PDT_W_SET_NAME = 23;
    public static final int PDT_C_DO_JOB = 16;
    public static final int PDT_C_GET_DATA = 17;
    public static final int PDT_RATELIMIT = 18;
    public static final int PDT_BREAKER = 19;
    public static final int PDT_SC_REG_SERVICE = 19;
    public static final int PDT_SC_OFF_SERVICE = 20;
    public static final int PDT_S_REG_SERVICE_OK = 21;

    // connect types & status
    public static final int CONN_TYPE_INIT = 0;
    public static final int CONN_TYPE_SERVER = 1;
    public static final int CONN_TYPE_WORKER = 2;
    public static final int CONN_TYPE_CLIENT = 3;
    public static final int CONN_TYPE_SERVICE = 4;
    public static final int PARAMS_TYPE_MSGPACK = 5;
    public static final int PARAMS_TYPE_JSON = 6;
    public static final int JOB_STATUS_INIT = 7;
    public static final int JOB_STATUS_DOING = 8;
    public static final int JOB_STATUS_DONE = 9;
    public static final int PARAMS_HANDLE_TYPE_ENCODE = 10;
    public static final int PARAMS_HANDLE_TYPE_ORIGINAL = 11;

    // loadbalance type
    public static final int LOADBALANCE_HASH = 1;
    public static final int LOADBALANCE_LRU = 2;
    public static final int LOADBALANCE_ROUND_WEIGHT = 3;

    // rate limit
    public static final int MAX_PENDING = 1000;
    public static final int MAX_OUTSTANDING = 10;
    public static final long TARGET_LATENCY = 5L;   // ms
    public static final long FILL_INTERVAL = 1000L; // ms
    public static final int CAPACITY = 1000;

    // http request markers
    public static final String N_REQUEST_TYPE = "N-NMID-RequestType";
    public static final String N_PARAMS_TYPE = "N-NMID-ParamsType";
    public static final String N_PARAMS_HANDLE_TYPE = "N-NMID-ParamsHandleType";
    public static final String N_MESSAGE_STATUS_TYPE = "N-NMID-MessageStatusType";
    public static final String N_ERROR_MESSAGE = "N-NMID-ErrorMessage";
    public static final String N_PDT_DATA_TYPE = "N-NMID-PdtDataType";
    public static final String N_FUNCTION_NAME = "N-NMID-FunctionName";
    public static final String ETCD_BASE_KEY = "nmid/";
    public static final String RESTIMEOUT = "RESTIMEOUT";
    public static final String HTTP_DO_WORK = "HTTP_DO_WORK";
    public static final String HTTP_ADD_SERVICE = "HTTP_ADD_SERVICE";
    public static final String PARAMSTYPE_MSGPACK = "5";
    public static final String PARAMSTYPE_JSON = "6";
    public static final String PARAMSHANDLETYPE_ENCODE = "10";
    public static final String PARAMSHANDLETYPEORIGINAL = "11";
}
