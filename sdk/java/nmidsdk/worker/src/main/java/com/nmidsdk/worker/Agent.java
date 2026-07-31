package com.nmidsdk.worker;

import com.nmidsdk.model.Constants;
import com.nmidsdk.model.utils.Utils;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.net.Socket;
import java.util.Deque;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentLinkedDeque;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;

/**
 * Worker agent: a single connection from a worker to one nmid server.
 * Mirrors sdk/node/lib/worker/agent.js, but uses a blocking socket with a
 * dedicated reader thread.
 */
public class Agent {
    public final String network;
    private final Object addr;
    public final Worker worker;

    public Socket conn;
    private OutputStream out;
    private InputStream in;

    public final Request req = new Request();
    public volatile long lastTime;

    private volatile boolean closed = false;
    private Thread readerThread;

    // FIFO of resolvers waiting for a PDT_OK confirmation from the server.
    private final Deque<CompletableFuture<Void>> okResolvers = new ConcurrentLinkedDeque<>();

    public Agent(String network, Object addr, Worker worker) {
        this.network = network == null ? "tcp" : network;
        this.addr = addr;
        this.worker = worker;
        this.lastTime = Utils.getMillisecond();
    }

    /**
     * Wait for the next PDT_OK from the server. The server sends PDT_OK in
     * response to PDT_W_SET_NAME and PDT_W_ADD_FUNC, so awaiting this after
     * each registration packet guarantees the server has actually processed it
     * before we proceed (closes the "have no job do" race for early client
     * calls). A safety timeout prevents hanging forever if the OK is lost.
     */
    public CompletableFuture<Void> expectOk(long timeoutMs) {
        CompletableFuture<Void> future = new CompletableFuture<>();
        okResolvers.add(future);

        ScheduledExecutorService scheduler = worker.getScheduler();
        if (scheduler != null) {
            scheduler.schedule(() -> {
                if (!future.isDone()) {
                    okResolvers.remove(future);
                    worker.emitWarning(new Exception("PDT_OK timeout, proceeding anyway"));
                    future.complete(null);
                }
            }, timeoutMs, TimeUnit.MILLISECONDS);
        }
        return future;
    }

    public void onOk() {
        CompletableFuture<Void> f = okResolvers.poll();
        if (f != null) {
            f.complete(null);
        }
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

    /** Connect to the server. Completes on success. */
    public CompletableFuture<Void> connect() {
        CompletableFuture<Void> future = new CompletableFuture<>();
        try {
            String[] hp = parseAddr();
            conn = new Socket();
            conn.connect(new InetSocketAddress(hp[0], Integer.parseInt(hp[1])), Constants.DIAL_TIME_OUT);
            conn.setTcpNoDelay(true);
            out = conn.getOutputStream();
            in = conn.getInputStream();
            closed = false;
            lastTime = Utils.getMillisecond();

            readerThread = new Thread(this::readerLoop, "nmid-agent-reader");
            readerThread.setDaemon(true);
            readerThread.start();

            future.complete(null);
        } catch (Exception e) {
            future.completeExceptionally(e);
        }
        return future;
    }

    /** Reconnect an existing agent after a connection drop. */
    public CompletableFuture<Void> reConnect() {
        if (conn != null) {
            try { conn.close(); } catch (Exception ignored) {}
            conn = null;
        }
        return connect();
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

                if (connType != Constants.CONN_TYPE_SERVER) continue;

                Response resp = Response.decode(dataType, dataLen, body);
                if (resp == null) continue;
                resp.agent = this;

                // Dispatch on the worker (reader thread is single per agent, so
                // ordering is preserved; job execution itself is offloaded).
                worker.handleResp(resp);
            }
        } catch (IOException e) {
            if (!closed) {
                worker.emitError(e);
            }
        }
    }

    // ---- atomic send helpers ------------------------------------------------
    // Each method builds the packet and writes it under the agent lock, so
    // concurrent writers (reader thread returning a job, heartbeat timer,
    // registration) never corrupt the shared Request.

    public synchronized void sendSetName(String name) {
        req.setWorkerName(name);
        rawWrite();
    }

    public synchronized void sendAddFunc(String name) {
        req.addFunctionPack(name);
        rawWrite();
    }

    public synchronized void sendDelFunc(String name) {
        req.delFunctionPack(name);
        rawWrite();
    }

    public synchronized void sendHeartbeat() {
        req.heartBeatPack();
        rawWrite();
    }

    public synchronized void sendGrabJob() {
        req.grabDataPack();
        rawWrite();
    }

    public synchronized void sendWakeup() {
        req.wakeupPack();
        rawWrite();
    }

    public synchronized void sendReturnData(Response job, byte[] ret) {
        req.handleLen = job.handleLen;
        req.handle = job.handle;
        req.paramsLen = job.paramsLen;
        req.params = job.params;
        req.jobIdLen = job.jobIdLen;
        req.jobId = job.jobId;
        req.retPack(ret);
        rawWrite();
    }

    private void rawWrite() {
        if (conn == null || closed) return;
        byte[] buf = req.encodePack();
        try {
            out.write(buf);
            out.flush();
        } catch (IOException e) {
            worker.emitError(e);
        }
    }

    public void heartBeatPing() {
        sendHeartbeat();
    }

    public void delOldFuncMsg(String funcName) {
        sendDelFunc(funcName);
    }

    public void wakeup() {
        sendWakeup();
    }

    public void close() {
        closed = true;
        Socket s = conn;
        if (s != null) {
            try { s.close(); } catch (Exception ignored) {}
            conn = null;
        }
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
