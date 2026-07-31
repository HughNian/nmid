package com.nmidsdk.model.utils;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.msgpack.jackson.dataformat.MessagePackFactory;

import java.nio.charset.StandardCharsets;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Shared utility helpers: id generation, msgpack/json packing, params decoding
 * and the standard RetStruct builder. Mirrors sdk/node/lib/worker/utils.js.
 */
public final class Utils {
    private Utils() {}

    private static final ObjectMapper MSGPACK = new ObjectMapper(new MessagePackFactory());
    private static final ObjectMapper JSON = new ObjectMapper();
    private static final AtomicLong ID_GEN = new AtomicLong(0);

    private static final TypeReference<Map<String, Object>> MAP_TYPE = new TypeReference<Map<String, Object>>() {};

    /** Monotonically increasing id, similar in spirit to the node IdGenerator. */
    public static String getId() {
        long value = System.nanoTime() << 32;
        long next = ID_GEN.incrementAndGet();
        return Long.toString(value + next);
    }

    public static long getMillisecond() {
        return System.currentTimeMillis();
    }

    public static long getNowSecond() {
        return System.currentTimeMillis() / 1000;
    }

    /** Encode a value to a msgpack byte[]. */
    public static byte[] msgpackPack(Object obj) {
        try {
            return MSGPACK.writeValueAsBytes(obj);
        } catch (Exception e) {
            throw new RuntimeException("msgpack encode failed", e);
        }
    }

    public static Map<String, Object> msgpackParamsMap(byte[] params) {
        if (params == null || params.length == 0) return null;
        try {
            return MSGPACK.readValue(params, MAP_TYPE);
        } catch (Exception e) {
            throw new RuntimeException("msgpack params decode failed", e);
        }
    }

    public static Map<String, Object> jsonParamsMap(byte[] params) {
        if (params == null || params.length == 0) return null;
        try {
            return JSON.readValue(params, MAP_TYPE);
        } catch (Exception e) {
            throw new RuntimeException("json params decode failed", e);
        }
    }

    /**
     * Build the standard nmid return struct { Code, Msg, Data } as a msgpack
     * byte[]. The Data field is encoded as a msgpack bin type, matching the
     * node/python/go encoders.
     */
    public static byte[] buildRetStruct(int code, String msg, byte[] data) {
        Map<String, Object> obj = new LinkedHashMap<>();
        obj.put("Code", code);
        obj.put("Msg", msg);
        obj.put("Data", data == null ? new byte[0] : data);
        return msgpackPack(obj);
    }

    /** Decode a msgpack-encoded RetStruct ({Code,Msg,Data}) from the ret field. */
    public static RetStruct decodeRet(byte[] ret) {
        if (ret == null || ret.length == 0) return null;
        try {
            Map<String, Object> m = MSGPACK.readValue(ret, MAP_TYPE);
            RetStruct r = new RetStruct();
            Object c = m.get("Code");
            r.Code = c == null ? 0 : ((Number) c).intValue();
            Object msg = m.get("Msg");
            r.Msg = msg == null ? "" : msg.toString();
            Object d = m.get("Data");
            if (d == null) {
                r.Data = new byte[0];
            } else if (d instanceof byte[]) {
                r.Data = (byte[]) d;
            } else if (d instanceof String) {
                r.Data = ((String) d).getBytes(StandardCharsets.UTF_8);
            } else {
                r.Data = new byte[0];
            }
            return r;
        } catch (Exception e) {
            throw new RuntimeException("decodeRet failed", e);
        }
    }
}
