package com.nmidsdk.worker;

import com.nmidsdk.model.Constants;
import com.nmidsdk.model.utils.Utils;

import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.Map;

/**
 * Worker response decoder.
 * Mirrors sdk/node/lib/worker/response.js.
 *
 * For PDT_S_GET_DATA the body (read from the packet at MIN_DATA_SIZE) is:
 *   ParamsType(4) | ParamsHandleType(4) | HandleLen(4) | ParamsLen(4) |
 *   JobIdLen(4) | Handle | Params | JobId
 * Params are parsed (msgpack or json) into paramsMap.
 */
public class Response {
    public int dataType;
    public byte[] data = new byte[0];
    public int dataLen;

    public String handle = "";
    public int handleLen;
    public int paramsType;
    public int paramsHandleType;
    public int paramsLen;
    public byte[] params = new byte[0];
    public Map<String, Object> paramsMap;
    public String jobId = "";
    public int jobIdLen;

    /** Back-reference to the Agent that received this job. */
    public Agent agent;

    public Response getResponse() {
        return this;
    }

    public byte[] getParams() {
        if (this.paramsLen == 0) return null;
        return this.params;
    }

    public Map<String, Object> getParamsMap() {
        if (this.paramsLen == 0) return null;
        return this.paramsMap;
    }

    public void parseParams(byte[] params) {
        this.params = params;
        if (this.paramsType == Constants.PARAMS_TYPE_MSGPACK) {
            this.paramsMap = Utils.msgpackParamsMap(params);
        } else if (this.paramsType == Constants.PARAMS_TYPE_JSON) {
            this.paramsMap = Utils.jsonParamsMap(params);
        }
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

        if (dataType == Constants.PDT_S_GET_DATA) {
            int start = 0;
            resp.paramsType = readIntBE(body, start);
            start += Constants.UINT32_SIZE;
            resp.paramsHandleType = readIntBE(body, start);
            start += Constants.UINT32_SIZE;
            resp.handleLen = readIntBE(body, start);
            start += Constants.UINT32_SIZE;
            resp.paramsLen = readIntBE(body, start);
            start += Constants.UINT32_SIZE;
            resp.jobIdLen = readIntBE(body, start);
            start += Constants.UINT32_SIZE;

            resp.handle = new String(body, start, resp.handleLen, StandardCharsets.UTF_8);
            start += resp.handleLen;

            byte[] paramsBytes = Arrays.copyOfRange(body, start, start + resp.paramsLen);
            resp.parseParams(paramsBytes);
            start += resp.paramsLen;

            resp.jobId = new String(body, start, resp.jobIdLen, StandardCharsets.UTF_8);
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
