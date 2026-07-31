package com.nmidsdk.worker;

import com.nmidsdk.model.Constants;

import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;

/**
 * Worker request encoder.
 * Mirrors sdk/node/lib/worker/request.js.
 *
 * Every worker packet starts with header (big-endian):
 *   ConnType(4)=CONN_TYPE_WORKER | DataType(4) | DataLen(4)
 * followed by DataLen bytes of body. Most control packets carry the body as
 * plain utf-8 bytes (function name / worker name / "PING"); the return-data
 * packet (PDT_W_RETURN_DATA) carries a structured body (see retPack).
 */
public class Request {
    public int dataType;
    public byte[] data = new byte[0];
    public int dataLen;

    public String handle = "";
    public int handleLen;
    public int paramsLen;
    public byte[] params = new byte[0];
    public String jobId = "";
    public int jobIdLen;
    public byte[] ret = new byte[0];
    public int retLen;

    private void setBytes(int dataType, byte[] payload) {
        this.dataType = dataType;
        this.data = payload;
        this.dataLen = payload.length;
    }

    public void heartBeatPack() {
        setBytes(Constants.PDT_W_HEARTBEAT_PING, "PING".getBytes(StandardCharsets.UTF_8));
    }

    public void setWorkerName(String workerName) {
        setBytes(Constants.PDT_W_SET_NAME, workerName.getBytes(StandardCharsets.UTF_8));
    }

    public void addFunctionPack(String funcName) {
        setBytes(Constants.PDT_W_ADD_FUNC, funcName.getBytes(StandardCharsets.UTF_8));
    }

    public void delFunctionPack(String funcName) {
        setBytes(Constants.PDT_W_DEL_FUNC, funcName.getBytes(StandardCharsets.UTF_8));
    }

    public void grabDataPack() {
        setBytes(Constants.PDT_W_GRAB_JOB, new byte[0]);
    }

    public void wakeupPack() {
        setBytes(Constants.PDT_WAKEUP, new byte[0]);
    }

    /**
     * Build the PDT_W_RETURN_DATA body, echoing back the original job's handle,
     * params and jobId together with the computed ret.
     *
     * body layout:
     *   HandleLen(4) | Handle | ParamsLen(4) | Params |
     *   RetLen(4) | Ret | JobIdLen(4) | JobId
     */
    public void retPack(byte[] ret) {
        this.ret = ret == null ? new byte[0] : ret;
        this.retLen = this.ret.length;

        this.dataType = Constants.PDT_W_RETURN_DATA;
        this.dataLen =
            Constants.UINT32_SIZE + this.handleLen +
            Constants.UINT32_SIZE + this.paramsLen +
            Constants.UINT32_SIZE + this.retLen +
            Constants.UINT32_SIZE + this.jobIdLen;

        ByteBuffer content = ByteBuffer.allocate(this.dataLen);
        content.putInt(this.handleLen);
        content.put(this.handle.getBytes(StandardCharsets.UTF_8));
        content.putInt(this.paramsLen);
        content.put(this.params);
        content.putInt(this.retLen);
        content.put(this.ret);
        content.putInt(this.jobIdLen);
        content.put(this.jobId.getBytes(StandardCharsets.UTF_8));

        this.data = content.array();
    }

    /** Prepend the 12-byte header (CONN_TYPE_WORKER). */
    public byte[] encodePack() {
        int total = Constants.MIN_DATA_SIZE + this.dataLen;
        ByteBuffer data = ByteBuffer.allocate(total);
        data.putInt(Constants.CONN_TYPE_WORKER);
        data.putInt(this.dataType);
        data.putInt(this.dataLen);
        if (this.dataLen > 0) {
            data.put(this.data);
        }
        return data.array();
    }
}
