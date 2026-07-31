package com.nmidsdk.example.client;

import com.nmidsdk.client.Client;
import com.nmidsdk.client.NmidError;
import com.nmidsdk.client.Response;
import com.nmidsdk.model.Constants;
import com.nmidsdk.model.utils.RetStruct;
import com.nmidsdk.model.utils.Utils;

import java.nio.charset.StandardCharsets;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CompletionException;

/**
 * nmid client example.
 * Mirrors sdk/node/example/client/testclient.js.
 *
 * Run a nmid server (127.0.0.1:6808) and a worker exposing "ToUpper" first,
 * then:
 *   mvn -q -pl example/example-client -am exec:java
 *      -Dexec.mainClass=com.nmidsdk.example.client.Main
 */
public class Main {
    private static final String SERVER_HOST = "127.0.0.1";
    private static final int SERVER_PORT = 6808;

    public static void main(String[] args) throws Exception {
        Client client = new Client("tcp", SERVER_HOST + ":" + SERVER_PORT);
        client.errHandler = e -> {
            if (e instanceof NmidError && "PDT_CANT_DO".equals(((NmidError) e).code)) {
                // transient: no worker registered for the function right now
                System.err.println("have no job do (transient, will retry)");
            } else {
                System.err.println("Client error: "
                    + (e.getMessage() == null ? e : e.getMessage()));
            }
        };

        try {
            client.start().join();
        } catch (CompletionException e) {
            Throwable cause = e.getCause() == null ? e : e.getCause();
            System.err.println("Error starting client: " + cause.getMessage());
            return;
        }

        Map<String, Object> params = new LinkedHashMap<>();
        params.put("name", "nihaonihao");
        byte[] paramsBuf = Utils.msgpackPack(params);

        try {
            Response resp = callWithRetry(client, "ToUpper", paramsBuf, 5, 300);
            handleResponse(resp);
        } catch (Exception e) {
            client.errHandler.accept(e);
        } finally {
            client.close();
        }
    }

    private static void handleResponse(Response resp) {
        if (resp == null
            || resp.dataType != Constants.PDT_S_RETURN_DATA
            || resp.retLen == 0) {
            return;
        }
        RetStruct retStruct = Utils.decodeRet(resp.ret);
        if (retStruct.Code != 0) {
            System.out.println(retStruct.Msg);
            return;
        }
        System.out.println(new String(retStruct.Data, StandardCharsets.UTF_8));
    }

    /**
     * doAsync rejects on PDT_CANT_DO when no worker is available yet. That is a
     * transient condition (worker still registering, or mid-reconnect), so
     * retry a few times before giving up.
     */
    private static Response callWithRetry(Client client, String funcName, byte[] params,
                                          int retries, long delayMs) throws Exception {
        Exception lastErr = null;
        for (int i = 0; i < retries; i++) {
            try {
                CompletableFuture<Response> f = client.doAsync(funcName, params);
                return f.join();
            } catch (CompletionException e) {
                Throwable cause = e.getCause() == null ? e : e.getCause();
                if (cause instanceof NmidError
                    && "PDT_CANT_DO".equals(((NmidError) cause).code)
                    && i < retries - 1) {
                    Thread.sleep(delayMs);
                    continue;
                }
                if (cause instanceof Exception) {
                    lastErr = (Exception) cause;
                    throw lastErr;
                }
                throw e;
            }
        }
        throw lastErr == null ? new Exception("call failed") : lastErr;
    }
}
