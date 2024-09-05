package nmid.worker;

import java.util.*;

import nmid.utils.Utils;
import nmid.consts.Constants;
import java.util.concurrent.locks.Lock;
import java.util.concurrent.locks.ReentrantLock;

/**
 * Worker class
 *
 */
public class Worker
{
    private Lock lock = new ReentrantLock();
    public String workerId;
    public String workerName;
    public List<Agent> agents = new ArrayList<>();
    public Map<String, Function> functions = new HashMap<>();
    public int funcsNum = 0;
    public volatile boolean ready = false;
    public volatile boolean running = false;
    public volatile boolean useTrace = false;

    public Worker SetWorkerId(String wid) {
        if (wid.length() == 0) {
            this.workerId = Utils.GetId();
        } else {
            this.workerId = wid;
        }

        return this;
    }

    public Worker SetWorkerName(String wname) {
        if (wname.length() == 0) {
            this.workerName = Utils.GetId();
        } else {
            this.workerName = wname;
        }

        return this;
    }

    public String GetWorkerKey() {
        String key = this.workerName;
        if (key.isEmpty()) {
            key = this.workerId;
        }

        if (key.isEmpty()) {
            key = Utils.GetId();
        }

        return key;
    }

    public void AddServer(String net, String addr) throws Exception {
        Agent agent = new Agent(net, addr, this);

        if (agent == null) {
            throw new Exception("agent is null");
        }

        this.agents.add(agent);
    }

    public synchronized void AddFunction(String funcName, Function jobFunc) throws Exception {
        lock.lock();
        try {
            if (this.functions.containsKey(funcName)) {
                throw new Exception("function already exist");
            }

            this.functions.put(funcName, jobFunc);
            this.funcsNum++;
        } finally {
            lock.unlock();
        }

        if (this.running) {
            new Thread(() -> {
                try {
                    this.MsgBroadcast(funcName, Constants.PDT_W_ADD_FUNC);
                } catch (Exception e) {
                    e.printStackTrace();
                }
            }).start();
        }
    }

    public synchronized void DelFunction(String funcName) throws Exception {
        lock.lock();
        try {
            if (!this.functions.containsKey(funcName)) {
                throw new Exception("function not exist");
            }

            this.functions.remove(funcName);
            this.funcsNum--;
        } finally {
            lock.unlock();
        }

        if (this.running) {
            new Thread(() -> {
                try {
                    this.MsgBroadcast(funcName, Constants.PDT_W_DEL_FUNC);
                } catch (Exception e) {
                    e.printStackTrace();
                }
            }).start();
        }
    }

    public void MsgBroadcast(String name, int flag) {
        for (Agent a : agents) {
            switch(flag) {
                case Constants.PDT_W_SET_NAME:
                    a.req.SetWorkerName(name);
                    break;
                case Constants.PDT_W_ADD_FUNC:
                    a.req.AddFunctionPack(name);
                    break;
                case Constants.PDT_W_DEL_FUNC:
                    a.req.DelFunctionPack(name);
                    break;
            }

            a.Write();
        }
    }
}
