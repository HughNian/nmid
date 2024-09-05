package com.nmidsdk.worker;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import com.nmidsdk.worker.consts.Constants;

/**
 * Request class for worker
 *
 */
public class Request {
    public int dataType;
    public byte[] data;
    public int dataLen;
    public String handle;
    public int paramsType;
    public byte[] params;
    public String jobId;
    public byte[] ret;

    public void SetWorkerName(String workerName) {
        this.dataType = Constants.PDT_W_SET_NAME;
        this.dataLen = workerName.getBytes().length;
        this.data = workerName.getBytes();
    }

    public void AddFunctionPack(String funcName) {
        this.dataType = Constants.PDT_W_ADD_FUNC;
        this.dataLen = funcName.getBytes().length;
        this.data = funcName.getBytes();
    }

    public void DelFunctionPack(String funcName) {
        this.dataType = Constants.PDT_W_DEL_FUNC;
        this.dataLen = funcName.getBytes().length;
        this.data = funcName.getBytes();
    }

    public byte[] GrabDataPack() {
        this.dataType = Constants.PDT_W_GRAB_JOB;
        this.dataLen  = 0;
        this.data = new byte[0];

        return this.data;
    }

    public byte[] EncodePack() {
        int len = Constants.MIN_DATA_SIZE + this.dataLen;
        ByteBuffer data = ByteBuffer.allocate(len);
        data.order(ByteOrder.BIG_ENDIAN);

        data.putInt(Constants.CONN_TYPE_WORKER);
        data.putInt(this.dataType);
        data.putInt(this.dataLen);
        data.position(Constants.MIN_DATA_SIZE);
        data.put(this.data);

        return data.array();
    }
}
