<?php
namespace Nmid;

// Constants for the nmid wire protocol.
// Mirrors sdk/node/lib/const.js, sdk/java/.../Constants.java, sdk/rust/model.

class Constants
{
    // sizes & timing
    public const MIN_DATA_SIZE = 12;     // packet header: ConnType(4)+DataType(4)+DataLen(4)
    public const UINT32_SIZE = 4;
    public const DIAL_TIME_OUT = 6;       // seconds, connect timeout
    public const DEFAULT_HEARTBEAT_TIME = 10;   // seconds
    public const NMID_SERVER_TIMEOUT = 60000;   // ms

    // package data type
    public const PDT_OK = 1;
    public const PDT_ERROR = 2;
    public const PDT_CAN_DO = 3;
    public const PDT_CANT_DO = 4;
    public const PDT_NO_JOB = 5;
    public const PDT_HAVE_JOB = 6;
    public const PDT_TOSLEEP = 7;
    public const PDT_WAKEUP = 8;
    public const PDT_WAKEUPED = 9;
    public const PDT_S_GET_DATA = 10;
    public const PDT_S_RETURN_DATA = 11;
    public const PDT_W_GRAB_JOB = 12;
    public const PDT_W_ADD_FUNC = 13;
    public const PDT_W_DEL_FUNC = 14;
    public const PDT_W_RETURN_DATA = 15;
    public const PDT_C_DO_JOB = 16;
    public const PDT_C_GET_DATA = 17;
    public const PDT_RATELIMIT = 18;
    public const PDT_W_HEARTBEAT_PING = 22;
    public const PDT_S_HEARTBEAT_PONG = 23;
    public const PDT_W_SET_NAME = 23;

    // connect types
    public const CONN_TYPE_INIT = 0;
    public const CONN_TYPE_SERVER = 1;
    public const CONN_TYPE_WORKER = 2;
    public const CONN_TYPE_CLIENT = 3;
    public const CONN_TYPE_SERVICE = 4;

    // params type & handle type
    public const PARAMS_TYPE_MSGPACK = 5;
    public const PARAMS_TYPE_JSON = 6;
    public const PARAMS_HANDLE_TYPE_ENCODE = 10;
    public const PARAMS_HANDLE_TYPE_ORIGINAL = 11;
}
