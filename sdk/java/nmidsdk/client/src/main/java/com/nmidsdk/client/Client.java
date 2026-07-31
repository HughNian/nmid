package com.nmidsdk.client;

import com.nmidsdk.model.Constants;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.util.Deque;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentLinkedDeque;
import java.util.function.Consumer;

/**
 * nmid client.
 * Mirrors sdk/node/lib/client/client.js, but uses a blocking socket with a
 * dedicated reader thread instead of node's event loop.
 *
 * Usage:
 *   Client client = new Client("tcp", "127.0.0.1:6808");
 *   client.start().join();
 *   Response resp = client.doAsync("ToUpper", paramsBuf).join();
 *   client.close();
 *
 * addr may be a String "host:port" or a String[]{host, port}.
 */
public class Client {
    private final String network;
    private final Object addr;

    private Socket conn;
    private OutputStream out;
    private InputStream in;

    private final Request req = new Request();
    private final Object writeLock = new Object();

    /** Optional global error sink, mirroring node's errHandler. */
    public Consumer<Throwable> errHandler;

    // handle -> queue of pending promises (FIFO per handle)
    private final Map<String, Deque<CompletableFuture<Response>>> pending = new ConcurrentHashMap<>();
    private volatile boolean closed = false;
    private Thread readerThread;

    public Client(String network, Object addr) {
        this.network = network == null ? "tcp" : network;
        this.addr = addr;
    }

    public Client setIoTimeOut(int t) {
        return this;
    }

    public Client setParamsHandle(int hType) {
        if (hType == Constants.PARAMS_HANDLE_TYPE_ENCODE
            || hType == Constants.PARAMS_HANDLE_TYPE_ORIGINAL) {
            req.paramsHandleType = hType;
        }
        return this;
    }

    private String[] parseAddr() {
        if (addr instanceof String[]) {
            String[] a = (String[]) addr;
            if (a.length < 2) throw new IllegalArgumentException("invalid addr array");
            return new String[]{a[0], a[1]};
        }
        if (addr instanceof String) {
            String s = (String) addr;
            int idx = s.lastIndexOf(':');
            if (idx == -1) throw new IllegalArgumentException("invalid addr: " + s);
            return new String[]{s.substring(0, idx), s.substring(idx + 1)};
        }
        throw new IllegalArgumentException("invalid addr: " + addr);
    }

    /** Connect to the nmid server. Completes with this client on success. */
    public CompletableFuture<Client> start() {
        CompletableFuture<Client> future = new CompletableFuture<>();
        try {
            String[] hp = parseAddr();
            String host = hp[0];
            int port = Integer.parseInt(hp[1]);

            conn = new Socket();
            conn.connect(new InetSocketAddress(host, port), Constants.DIAL_TIME_OUT);
            conn.setTcpNoDelay(true);
            out = conn.getOutputStream();
            in = conn.getInputStream();
            closed = false;

            readerThread = new Thread(this::readerLoop, "nmid-client-reader");
            readerThread.setDaemon(true);
            readerThread.start();

            future.complete(this);
        } catch (Exception e) {
            future.completeExceptionally(e);
        }
        return future;
    }

    private void readerLoop() {
        try {
            while (!closed) {
                byte[] header = readN(in, Constants.MIN_DATA_SIZE);
                if (header == null) break;

                int connType = Response.readIntBE(header, 0);
                int dataType = Response.readIntBE(header, 4);
                int dataLen = Response.readIntBE(header, 8);

                byte[] body = readN(in, dataLen);
                if (body == null) break;

                // ignore packets not coming from the server
                if (connType != Constants.CONN_TYPE_SERVER) continue;

                Response resp = Response.decode(dataType, dataLen, body);
                if (resp == null) continue;
                processResp(resp);
            }
        } catch (IOException e) {
            if (!closed) {
                onError(e);
            }
        }
        // connection ended: fail anything still pending
        failAll(new IOException("connection closed"));
    }

    private void processResp(Response resp) {
        int dt = resp.dataType;
        if (dt == Constants.PDT_ERROR || dt == Constants.PDT_CANT_DO || dt == Constants.PDT_RATELIMIT) {
            NmidError err = resp.getResError();
            failAll(err);
            if (errHandler != null) errHandler.accept(err);
            return;
        }
        if (dt == Constants.PDT_S_RETURN_DATA) {
            handleResp(resp);
        }
    }

    private void handleResp(Response resp) {
        if (resp.handleLen == 0) return;
        Deque<CompletableFuture<Response>> q = pending.get(resp.handle);
        if (q != null) {
            CompletableFuture<Response> f = q.poll();
            if (f != null) {
                f.complete(resp);
            }
        }
    }

    private void failAll(Throwable err) {
        for (Deque<CompletableFuture<Response>> q : pending.values()) {
            CompletableFuture<Response> f;
            while ((f = q.poll()) != null) {
                f.completeExceptionally(err);
            }
        }
        pending.clear();
    }

    private void send(String funcName, byte[] params) throws IOException {
        synchronized (writeLock) {
            req.contentPack(Constants.PDT_C_DO_JOB, funcName, params);
            byte[] buf = req.encodePack();
            out.write(buf);
            out.flush();
        }
    }

    /**
     * Promise-style call.
     *   Response resp = client.doAsync("ToUpper", paramsBuf).join();
     */
    public CompletableFuture<Response> doAsync(String funcName, byte[] params) {
        if (conn == null || closed) {
            CompletableFuture<Response> f = new CompletableFuture<>();
            f.completeExceptionally(new IllegalStateException("Connection is not established"));
            return f;
        }
        CompletableFuture<Response> future = new CompletableFuture<>();
        pending.computeIfAbsent(funcName, k -> new ConcurrentLinkedDeque<>()).add(future);
        try {
            send(funcName, params);
        } catch (IOException e) {
            Deque<CompletableFuture<Response>> q = pending.get(funcName);
            if (q != null) q.remove(future);
            future.completeExceptionally(e);
        }
        return future;
    }

    /**
     * Callback-style call, mirroring the python/node API:
     *   client.doCall("ToUpper", paramsBuf, resp -> { ... });
     */
    public void doCall(String funcName, byte[] params, Consumer<Response> callback) {
        doAsync(funcName, params).whenComplete((resp, err) -> {
            if (err != null) {
                if (errHandler != null) errHandler.accept(err);
            } else if (callback != null) {
                callback.accept(resp);
            }
        });
    }

    private void onError(Throwable err) {
        if (errHandler != null) errHandler.accept(err);
        failAll(err);
    }

    public void close() {
        closed = true;
        Socket s = conn;
        if (s != null) {
            try { s.close(); } catch (IOException ignored) {}
            conn = null;
        }
        failAll(new IOException("client closed"));
    }

    /** Read exactly n bytes, blocking; returns null on EOF. */
    private static byte[] readN(InputStream in, int n) throws IOException {
        if (n == 0) return new byte[0];
        byte[] buf = new byte[n];
        int off = 0;
        while (off < n) {
            int r = in.read(buf, off, n - off);
            if (r < 0) return null;
            off += r;
        }
        return buf;
    }
}
