package com.nmidsdk.client;

import com.nmidsdk.model.Constants;

import java.nio.ByteBuffer;
import java.nio.charset.StandardCharsets;

/**
 * Client request encoder.
 * Mirrors sdk/node/lib/client/request.js.
 *
 * Packet layout (big-endian):
 *   header (12 bytes): ConnType(4) | DataType(4) | DataLen(4)
 *   body (DataLen bytes) for PDT_C_DO_JOB:
 *     ParamsType(4) | ParamsHandleType(4) | HandleLen(4) | Handle |
 *     ParamsLen(4) | Params
 */
public class Request {
    public int dataType;
    public byte[] data = new byte[0];
    public int dataLen;

    public String handle = "";
    public int handleLen;
    public int paramsType = Constants.PARAMS_TYPE_MSGPACK;
    public int paramsHandleType = Constants.PARAMS_HANDLE_TYPE_ENCODE;
    public int paramsLen;
    public byte[] params = new byte[0];

    /** Build the body for a client do-job request. */
    public void contentPack(int dataType, String handle, byte[] params) {
        this.dataType = dataType;
        this.handle = handle;
        byte[] handleBytes = handle.getBytes(StandardCharsets.UTF_8);
        this.handleLen = handleBytes.length;
        this.params = params == null ? new byte[0] : params;
        this.paramsLen = this.params.length;
        this.dataLen =
            Constants.UINT32_SIZE + // paramsType
            Constants.UINT32_SIZE + // paramsHandleType
            Constants.UINT32_SIZE + // handleLen
            this.handleLen +
            Constants.UINT32_SIZE + // paramsLen
            this.paramsLen;

        ByteBuffer content = ByteBuffer.allocate(this.dataLen);
        content.putInt(this.paramsType);
        content.putInt(this.paramsHandleType);
        content.putInt(this.handleLen);
        content.put(handleBytes);
        content.putInt(this.paramsLen);
        content.put(this.params);

        this.data = content.array();
    }

    /** Prepend the 12-byte header (CONN_TYPE_CLIENT). */
    public byte[] encodePack() {
        int total = Constants.MIN_DATA_SIZE + this.dataLen;
        ByteBuffer data = ByteBuffer.allocate(total);
        data.putInt(Constants.CONN_TYPE_CLIENT);
        data.putInt(this.dataType);
        data.putInt(this.dataLen);
        if (this.dataLen > 0) {
            data.put(this.data);
        }
        return data.array();
    }
}
