package com.nmidsdk.client;

import com.nmidsdk.model.Constants;

import java.nio.charset.StandardCharsets;
import java.util.Arrays;

/**
 * Client response decoder.
 * Mirrors sdk/node/lib/client/response.js.
 *
 * For PDT_S_RETURN_DATA the body (read from the packet at MIN_DATA_SIZE) is:
 *   HandleLen(4) | ParamsLen(4) | RetLen(4) | Handle | Params | Ret
 */
public class Response {
    public int dataType;
    public byte[] data = new byte[0];
    public int dataLen;

    public String handle = "";
    public int handleLen;
    public int paramsLen;
    public byte[] params = new byte[0];
    public int retLen;
    public byte[] ret = new byte[0];

    public NmidError getResError() {
        switch (this.dataType) {
            case Constants.PDT_ERROR:
                return new NmidError("PDT_ERROR", "request error");
            case Constants.PDT_CANT_DO:
                return new NmidError("PDT_CANT_DO", "have no job do");
            case Constants.PDT_RATELIMIT:
                return new NmidError("PDT_RATELIMIT", "have ratelimit");
            default:
                return null;
        }
    }

    public byte[] getResResult() {
        if (this.dataType == Constants.PDT_S_RETURN_DATA) {
            return this.ret;
        }
        return null;
    }

    /**
     * Decode a single complete packet body into a Response.
     *
     * @param dataType the packet's data type (from the header)
     * @param dataLen  the packet's content length (from the header)
     * @param body     the body bytes (length == dataLen), WITHOUT the 12-byte header
     * @return the decoded Response, or null if the body is incomplete
     */
    public static Response decode(int dataType, int dataLen, byte[] body) {
        if (body == null || body.length < dataLen) {
            return null;
        }

        Response resp = new Response();
        resp.dataType = dataType;
        resp.dataLen = dataLen;
        resp.data = body;

        if (dataType == Constants.PDT_S_RETURN_DATA) {
            int start = 0;
            resp.handleLen = readIntBE(body, start);
            start += Constants.UINT32_SIZE;
            resp.paramsLen = readIntBE(body, start);
            start += Constants.UINT32_SIZE;
            resp.retLen = readIntBE(body, start);
            start += Constants.UINT32_SIZE;

            resp.handle = new String(body, start, resp.handleLen, StandardCharsets.UTF_8);
            start += resp.handleLen;

            resp.params = Arrays.copyOfRange(body, start, start + resp.paramsLen);
            start += resp.paramsLen;

            resp.ret = Arrays.copyOfRange(body, start, start + resp.retLen);
        }

        return resp;
    }

    static int readIntBE(byte[] b, int off) {
        return ((b[off] & 0xFF) << 24)
            | ((b[off + 1] & 0xFF) << 16)
            | ((b[off + 2] & 0xFF) << 8)
            | (b[off + 3] & 0xFF);
    }
}
