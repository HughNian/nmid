package com.nmidsdk.worker;

import com.nmidsdk.model.Constants;
import com.nmidsdk.model.utils.Utils;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CompletionException;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.ThreadFactory;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

/**
 * nmid worker.
 * Mirrors sdk/node/lib/worker/worker.js.
 * <p>
 * A Worker holds one or more {@link Agent}s (connections to nmid servers) and
 * a set of registered functions. When a server pushes a job (PDT_S_GET_DATA)
 * the worker looks up the function by name, runs it on a job pool, and returns
 * the ret payload via PDT_W_RETURN_DATA. A heartbeat ping is sent on a timer.
 */
public class Worker {
    public String workerId = "";
    public String workerName = "";
    public final List<Agent> agents = new ArrayList<>();
    public final Map<String, Function> funcs = new ConcurrentHashMap<>();
    public int funcsNum = 0;

    public volatile boolean ready = false;
    public volatile boolean running = false;

    private ScheduledExecutorService scheduler;
    private ExecutorService workPool;

    private final List<Consumer<Throwable>> errorListeners = new CopyOnWriteArrayList<>();
    private final List<Consumer<Throwable>> warningListeners = new CopyOnWriteArrayList<>();

    ScheduledExecutorService getScheduler() {
        return scheduler;
    }

    /** Register a listener. Supported events: "error", "warning". */
    public Worker on(String event, Consumer<Throwable> cb) {
        if ("error".equals(event)) {
            errorListeners.add(cb);
        } else if ("warning".equals(event)) {
            warningListeners.add(cb);
        }
        return this;
    }

    public void emitError(Throwable err) {
        for (Consumer<Throwable> cb : errorListeners) {
            try { cb.accept(err); } catch (Exception ignored) {}
        }
    }

    public void emitWarning(Throwable err) {
        for (Consumer<Throwable> cb : warningListeners) {
            try { cb.accept(err); } catch (Exception ignored) {}
        }
    }

    public Worker setWorkerId(String wid) {
        this.workerId = (wid == null || wid.isEmpty()) ? Utils.getId() : wid;
        return this;
    }

    public Worker setWorkerName(String wname) {
        this.workerName = (wname == null || wname.isEmpty()) ? Utils.getId() : wname;
        return this;
    }

    public String getWorkerKey() {
        if (this.workerName != null && !this.workerName.isEmpty()) return this.workerName;
        if (this.workerId != null && !this.workerId.isEmpty()) return this.workerId;
        return Utils.getId();
    }

    /** network: "tcp"; addr: String[]{host, port} or "host:port". */
    public Worker addServer(String network, Object addr) {
        ensureExecutors();
        Agent agent = new Agent(network, addr, this);
        this.agents.add(agent);
        return this;
    }

    public synchronized void addFunction(String funcName, Function jobFunc) {
        if (this.funcs.containsKey(funcName)) {
            emitError(new Exception("function " + funcName + " already exist"));
            return;
        }
        this.funcs.put(funcName, jobFunc);
        this.funcsNum++;

        if (this.running) {
            msgBroadcast(funcName, Constants.PDT_W_ADD_FUNC);
        }
    }

    public synchronized void delFunction(String funcName) {
        if (!this.funcs.containsKey(funcName)) {
            emitError(new Exception("function " + funcName + " not exist"));
            return;
        }
        this.funcs.remove(funcName);
        this.funcsNum--;

        if (this.running) {
            msgBroadcast(funcName, Constants.PDT_W_DEL_FUNC);
        }
    }

    public Function getFunction(String funcName) {
        if (this.funcsNum == 0 || this.funcs.isEmpty()) return null;
        Function f = this.funcs.get(funcName);
        if (f == null) return null;
        return f;
    }

    /** Send a control message to every agent. */
    public void msgBroadcast(String name, int flag) {
        for (Agent agent : this.agents) {
            if (flag == Constants.PDT_W_SET_NAME) {
                agent.sendSetName(name);
            } else if (flag == Constants.PDT_W_ADD_FUNC) {
                agent.sendAddFunc(name);
            } else if (flag == Constants.PDT_W_DEL_FUNC) {
                agent.sendDelFunc(name);
            } else {
                agent.sendAddFunc(name);
            }
        }
    }

    /**
     * Connect all agents, set worker name and register all functions.
     * Waits for the server's PDT_OK confirmation after each SET_NAME / ADD_FUNC
     * so that early client calls don't hit PDT_CANT_DO ("have no job do").
     */
    public CompletableFuture<Void> workerReady() {
        if (this.agents.isEmpty()) {
            return failedFuture(new Exception("none active agents"));
        }
        if (this.funcsNum == 0 || this.funcs.isEmpty()) {
            return failedFuture(new Exception("none funcs"));
        }

        ensureExecutors();

        CompletableFuture<Void> chain = CompletableFuture.completedFuture(null);
        for (Agent agent : this.agents) {
            final Agent a = agent;
            chain = chain.thenCompose(v -> a.connect());
        }

        List<String> funcNames = new ArrayList<>(this.funcs.keySet());
        for (Agent agent : this.agents) {
            final Agent a = agent;
            chain = chain.thenCompose(v -> {
                a.sendSetName(this.workerName);
                return a.expectOk(5000);
            });
            for (final String fname : funcNames) {
                chain = chain.thenCompose(v -> {
                    a.sendAddFunc(fname);
                    return a.expectOk(5000);
                });
            }
        }

        chain = chain.thenRun(() -> this.ready = true);
        return chain;
    }

    /** Dispatch an incoming decoded response to the right handler. */
    public void handleResp(Response resp) {
        if (resp == null) return;
        int dt = resp.dataType;

        if (dt == Constants.PDT_OK) {
            // Server confirmed a SET_NAME / ADD_FUNC / DEL_FUNC packet.
            if (resp.agent != null) resp.agent.onOk();
            return;
        }

        if (dt == Constants.PDT_TOSLEEP) {
            final Agent a = resp.agent;
            if (scheduler != null && a != null) {
                scheduler.schedule(() -> a.wakeup(), 2000, TimeUnit.MILLISECONDS);
            }
            return;
        }

        if (dt == Constants.PDT_S_GET_DATA) {
            doFunction(resp);
            return;
        }

        if (dt == Constants.PDT_NO_JOB) {
            // server has no job right now; nothing to do
            return;
        }

        if (dt == Constants.PDT_S_HEARTBEAT_PONG) {
            if (resp.agent != null) resp.agent.lastTime = Utils.getMillisecond();
            return;
        }

        if (dt == Constants.PDT_WAKEUPED) {
            // woken up; server will push jobs when available
            return;
        }
    }

    /** Execute the registered function for a job and return its ret to the server. */
    public void doFunction(Response resp) {
        if (resp.dataType != Constants.PDT_S_GET_DATA) {
            emitError(new Exception("not get data"));
            return;
        }

        final String funcName = resp.handle;
        final Function fn = getFunction(funcName);
        if (fn == null) {
            emitError(new Exception("function " + funcName + " not found"));
            return;
        }
        if (resp.paramsLen == 0) {
            emitError(new Exception("params error"));
            return;
        }

        final Agent agent = resp.agent;
        final Response job = resp;
        workPool.submit(() -> {
            try {
                byte[] ret = fn.run(job);
                if (ret == null) {
                    emitError(new IllegalStateException("job function must return a byte[]"));
                    return;
                }
                agent.sendReturnData(job, ret);
            } catch (Exception e) {
                emitError(e);
            }
        });
    }

    /** Heartbeat ping on a fixed interval. */
    public void heartBeat() {
        if (scheduler == null) return;
        scheduler.scheduleAtFixedRate(() -> {
            for (Agent agent : this.agents) {
                agent.heartBeatPing();
            }
        }, Constants.DEFAULT_HEARTBEAT_TIME, Constants.DEFAULT_HEARTBEAT_TIME, TimeUnit.MILLISECONDS);
    }

    /** Periodically check for stale agents and reconnect them. */
    public void workerTimeOut() {
        if (scheduler == null) return;
        scheduler.scheduleAtFixedRate(() -> {
            for (Agent agent : this.agents) {
                // Reconnect only when the server has been truly unreachable for
                // a while. The threshold must be comfortably larger than the
                // heartbeat interval (10s): using 10s here caused spurious
                // reconnects at the heartbeat boundary (PONG not yet back),
                // which made clients intermittently see PDT_CANT_DO.
                if (Utils.getMillisecond() - agent.lastTime > Constants.NMID_SERVER_TIMEOUT) {
                    workerReConnect(agent);
                }
            }
        }, 5000, 5000, TimeUnit.MILLISECONDS);
    }

    /** Re-register and wait for PDT_OK, same as workerReady. */
    public void workerReConnect(final Agent agent) {
        // run on a separate thread so the scheduler is not blocked
        new Thread(() -> {
            try {
                for (String fname : this.funcs.keySet()) {
                    agent.delOldFuncMsg(fname);
                }
                agent.reConnect().join();

                agent.sendSetName(this.workerName);
                agent.expectOk(5000).join();
                for (String fname : this.funcs.keySet()) {
                    agent.sendAddFunc(fname);
                    agent.expectOk(5000).join();
                }
            } catch (CompletionException ce) {
                emitError(ce.getCause());
            } catch (Exception e) {
                emitError(e);
            }
        }, "nmid-reconnect").start();
    }

    /** Start the worker: connect, register, then run heartbeat + timeout loops. */
    public CompletableFuture<Void> workerDo() {
        if (!this.ready) {
            return workerReady().thenRun(() -> {
                this.running = true;
                heartBeat();
                workerTimeOut();
            });
        }
        this.running = true;
        heartBeat();
        workerTimeOut();
        return CompletableFuture.completedFuture(null);
    }

    public void workerClose() {
        if (this.running) {
            for (String fn : this.funcs.keySet()) {
                msgBroadcast(fn, Constants.PDT_W_DEL_FUNC);
            }
            for (Agent agent : this.agents) {
                agent.close();
            }
            this.running = false;
        }

        if (scheduler != null) {
            scheduler.shutdownNow();
            scheduler = null;
        }
        if (workPool != null) {
            workPool.shutdownNow();
            workPool = null;
        }
    }

    private synchronized void ensureExecutors() {
        if (scheduler == null) {
            scheduler = Executors.newScheduledThreadPool(2, daemonFactory("nmid-scheduler"));
        }
        if (workPool == null) {
            workPool = Executors.newCachedThreadPool(daemonFactory("nmid-work"));
        }
    }

    private static ThreadFactory daemonFactory(final String prefix) {
        ThreadFactory defaultFactory = Executors.defaultThreadFactory();
        return r -> {
            Thread t = defaultFactory.newThread(r);
            t.setDaemon(true);
            t.setName(prefix + "-" + t.getId());
            return t;
        };
    }

    private static <T> CompletableFuture<T> failedFuture(Throwable err) {
        CompletableFuture<T> f = new CompletableFuture<>();
        f.completeExceptionally(err);
        return f;
    }
}
