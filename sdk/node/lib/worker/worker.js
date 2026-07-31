'use strict';

// nmid worker.
// Mirrors sdk/python3/nmidsdk/worker/worker.py.
//
// A Worker holds one or more Agents (connections to nmid servers) and a set
// of registered functions. When a server pushes a job (PDT_S_GET_DATA) the
// worker looks up the function by name, runs it, and returns the ret payload
// via PDT_W_RETURN_DATA. A heartbeat ping is sent on a timer.

const { EventEmitter } = require('events');
const Agent = require('./agent');
const Function = require('./function');
const { IdGenerator, getMillisecond } = require('./utils');
const C = require('../const');

class Worker extends EventEmitter {
  constructor() {
    super();
    this.workerId = '';
    this.workerName = '';
    this.agents = [];
    this.funcs = new Map();
    this.funcsNum = 0;
    this.ready = false;
    this.running = false;
    this._heartbeatTimer = null;
    this._timeoutTimer = null;
  }

  setWorkerId(wid) {
    this.workerId = wid ? wid : new IdGenerator().get_id();
    return this;
  }

  setWorkerName(wname) {
    this.workerName = wname ? wname : new IdGenerator().get_id();
    return this;
  }

  // snake_case aliases for python parity
  set_worker_id(wid) { return this.setWorkerId(wid); }
  set_worker_name(wname) { return this.setWorkerName(wname); }

  getWorkerKey() {
    if (this.workerName) return this.workerName;
    if (this.workerId) return this.workerId;
    return new IdGenerator().get_id();
  }

  // network: 'tcp'; addr: [host, port] or 'host:port'
  addServer(network, addr) {
    const agent = new Agent(network, addr, this);
    this.agents.push(agent);
    return null;
  }

  addFunction(funcName, jobFunc) {
    if (this.funcs.has(funcName)) {
      const err = new Error(`function ${funcName} already exist`);
      this.emit('error', err);
      return err;
    }
    this.funcs.set(funcName, new Function(jobFunc, funcName));
    this.funcsNum += 1;

    if (this.running) {
      this.msgBroadcast(funcName, C.PDT_W_ADD_FUNC);
    }
    return null;
  }

  delFunction(funcName) {
    if (!this.funcs.has(funcName)) {
      const err = new Error(`function ${funcName} not exist`);
      this.emit('error', err);
      return err;
    }
    this.funcs.delete(funcName);
    this.funcsNum -= 1;

    if (this.running) {
      this.msgBroadcast(funcName, C.PDT_W_DEL_FUNC);
    }
    return null;
  }

  // snake_case aliases
  add_function(funcName, jobFunc) { return this.addFunction(funcName, jobFunc); }
  del_function(funcName) { return this.delFunction(funcName); }
  add_server(network, addr) { return this.addServer(network, addr); }

  getFunction(funcName) {
    if (this.funcsNum === 0 || this.funcs.size === 0) return null;
    const f = this.funcs.get(funcName);
    if (!f) return null;
    if (f.funcName !== funcName) return null;
    return f;
  }

  // Send a control message to every agent.
  msgBroadcast(name, flag) {
    for (const agent of this.agents) {
      if (flag === C.PDT_W_SET_NAME) {
        agent.req.setWorkerName(name);
      } else if (flag === C.PDT_W_ADD_FUNC) {
        agent.req.addFunctionPack(name);
      } else if (flag === C.PDT_W_DEL_FUNC) {
        agent.req.delFunctionPack(name);
      } else {
        agent.req.addFunctionPack(name);
      }
      agent.write();
    }
  }

  // Connect all agents, set worker name and register all functions.
  async workerReady() {
    if (this.agents.length === 0) {
      return new Error('none active agents');
    }
    if (this.funcsNum === 0 || this.funcs.size === 0) {
      return new Error('none funcs');
    }

    for (const agent of this.agents) {
      try {
        await agent.connect();
      } catch (e) {
        return e;
      }
    }

    // Register with each server and WAIT for its PDT_OK confirmation before
    // declaring ready. The server returns PDT_CANT_DO to clients for as long
    // as a function is not registered, so proceeding before the OK is
    // processed opens a race where early client calls get "have no job do".
    const funcNames = Array.from(this.funcs.keys());
    for (const agent of this.agents) {
      agent.req.setWorkerName(this.workerName);
      agent.write();
      await agent.expectOk();

      for (const fname of funcNames) {
        agent.req.addFunctionPack(fname);
        agent.write();
        await agent.expectOk();
      }
    }

    this.ready = true;
    return null;
  }

  // Dispatch an incoming decoded response to the right handler.
  async handleResp(resp) {
    if (!resp) return;
    const dt = resp.dataType;

    if (dt === C.PDT_OK) {
      // Server confirmed a SET_NAME / ADD_FUNC / DEL_FUNC packet.
      if (resp.agent) resp.agent.onOk();
      return;
    }

    if (dt === C.PDT_TOSLEEP) {
      setTimeout(() => {
        if (resp.agent) resp.agent.wakeup();
      }, 2000);
      return;
    }

    if (dt === C.PDT_S_GET_DATA) {
      const err = await this.doFunction(resp);
      if (err) this.emit('error', err);
      return;
    }

    if (dt === C.PDT_NO_JOB) {
      // server has no job right now; nothing to do (no active grab loop)
      return;
    }

    if (dt === C.PDT_S_HEARTBEAT_PONG) {
      if (resp.agent) resp.agent.lastTime = getMillisecond();
      return;
    }

    if (dt === C.PDT_WAKEUPED) {
      // woken up; server will push jobs when available
      return;
    }
  }

  // Execute the registered function for a job and return its ret to the server.
  async doFunction(resp) {
    if (resp.dataType !== C.PDT_S_GET_DATA) {
      return new Error('not get data');
    }

    const funcName = resp.handle;
    const fn = this.getFunction(funcName);
    if (!fn) {
      return new Error(`function ${funcName} not found`);
    }
    if (resp.paramsLen === 0) {
      return new Error('params error');
    }

    const agent = resp.agent;
    try {
      const ret = await fn.func(resp);
      if (!Buffer.isBuffer(ret)) {
        return new Error('job function must return a Buffer');
      }

      // Echo back the original job's handle / params / jobId.
      agent.req.handleLen = resp.handleLen;
      agent.req.handle = resp.handle;
      agent.req.paramsLen = resp.paramsLen;
      agent.req.params = resp.params;
      agent.req.jobIdLen = resp.jobIdLen;
      agent.req.jobId = resp.jobId;

      agent.req.retPack(ret);
      agent.write();
      return null;
    } catch (e) {
      return e;
    }
  }

  // Heartbeat ping on a fixed interval.
  heartBeat() {
    if (this._heartbeatTimer) {
      clearInterval(this._heartbeatTimer);
    }
    this._heartbeatTimer = setInterval(() => {
      for (const agent of this.agents) {
        agent.heartBeatPing();
      }
    }, C.DEFAULT_HEARTBEAT_TIME * 1000);
    if (this._heartbeatTimer.unref) this._heartbeatTimer.unref();
  }

  // Periodically check for stale agents and reconnect them.
  workerTimeOut() {
    if (this._timeoutTimer) {
      clearInterval(this._timeoutTimer);
    }
    this._timeoutTimer = setInterval(() => {
      for (const agent of this.agents) {
        // Reconnect only when the server has been truly unreachable for a
        // while. The threshold must be comfortably larger than the heartbeat
        // interval (10s): using 10s here caused spurious reconnects at the
        // heartbeat boundary (PONG not yet back), which torn down the
        // connection mid-flight and made clients intermittently see
        // PDT_CANT_DO ("have no job do"). The go worker uses
        // NMID_SERVER_TIMEOUT (60s) for the same reason.
        if (getMillisecond() - agent.lastTime > C.NMID_SERVER_TIMEOUT) {
          this.workerReConnect(agent).catch((e) => this.emit('error', e));
        }
      }
    }, 5000);
    if (this._timeoutTimer.unref) this._timeoutTimer.unref();
  }

  async workerReConnect(agent) {
    for (const fname of this.funcs.keys()) {
      agent.delOldFuncMsg(fname);
    }
    await agent.reConnect();

    // Re-register and wait for PDT_OK, same as workerReady, so that client
    // calls don't hit PDT_CANT_DO during the re-registration window.
    agent.req.setWorkerName(this.workerName);
    agent.write();
    await agent.expectOk();
    for (const fname of this.funcs.keys()) {
      agent.req.addFunctionPack(fname);
      agent.write();
      await agent.expectOk();
    }
  }

  // Start the worker: connect, register, then run heartbeat + timeout loops.
  async workerDo() {
    if (!this.ready) {
      const e = await this.workerReady();
      if (e) {
        this.emit('error', e);
        return;
      }
    }

    this.running = true;
    this.heartBeat();
    this.workerTimeOut();
  }

  workerClose() {
    if (this.running) {
      for (const fn of this.funcs.keys()) {
        this.msgBroadcast(fn, C.PDT_W_DEL_FUNC);
      }
      for (const agent of this.agents) {
        agent.close();
      }
      this.running = false;
    }

    if (this._heartbeatTimer) {
      clearInterval(this._heartbeatTimer);
      this._heartbeatTimer = null;
    }
    if (this._timeoutTimer) {
      clearInterval(this._timeoutTimer);
      this._timeoutTimer = null;
    }
  }

  // snake_case aliases
  worker_ready() { return this.workerReady(); }
  worker_do() { return this.workerDo(); }
  worker_close() { return this.workerClose(); }
}

module.exports = Worker;
