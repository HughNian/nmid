package com.nmidsdk.example.worker;

import com.nmidsdk.model.utils.Utils;
import com.nmidsdk.worker.Worker;

import java.nio.charset.StandardCharsets;
import java.util.Map;
import java.util.concurrent.CompletionException;

/**
 * nmid worker example.
 * Mirrors sdk/node/example/worker/worker1.js.
 *
 * Run a nmid server (127.0.0.1:6808) first, then:
 *   mvn -q -pl example/example-worker -am exec:java
 *      -Dexec.mainClass=com.nmidsdk.example.worker.Main
 */
public class Main {
    public static void main(String[] args) throws Exception {
        Worker worker = new Worker();
        worker.setWorkerName("Worker1");
        worker.addServer("tcp", "127.0.0.1:6808");

        // jobFunc receives the decoded Response and must return a byte[]
        // (the ret payload, typically a msgpack-encoded RetStruct).
        worker.addFunction("ToUpper", resp -> {
            Map<String, Object> paramsMap = resp.getParamsMap();
            if (paramsMap == null || paramsMap.get("name") == null) {
                throw new Exception("params error");
            }
            String name = String.valueOf(paramsMap.get("name"));
            byte[] data = name.toUpperCase().getBytes(StandardCharsets.UTF_8);
            return Utils.buildRetStruct(0, "ok", data);
        });

        worker.on("error", err -> System.err.println(
            "worker error: " + (err.getMessage() == null ? err : err.getMessage())));

        try {
            worker.workerReady().join();
        } catch (CompletionException e) {
            Throwable cause = e.getCause() == null ? e : e.getCause();
            System.err.println("worker ready error: " + cause.getMessage());
            worker.workerClose();
            return;
        }

        worker.workerDo();
        System.out.println("worker started, funcs:ToUpper");

        // graceful shutdown on SIGINT/SIGTERM
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            worker.workerClose();
        }));

        // keep the main thread alive; daemon threads handle IO / heartbeat
        Thread.currentThread().join();
    }
}
