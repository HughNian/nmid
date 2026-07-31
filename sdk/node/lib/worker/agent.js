'use strict';

// Worker agent: a single connection from a worker to one nmid server.
// Mirrors sdk/python3/nmidsdk/worker/agent.py, but uses Node's event-driven
// socket so no read thread is required.

const net = require('net');
const { EventEmitter } = require('events');
const Request = require('./request');
const { decodePack } = require('./response');
const { getMillisecond } = require('./utils');
const C = require('../const');

class Agent extends EventEmitter {
  // worker: the owning Worker instance
  constructor(network, addr, worker) {
    super();
    this.network = network || 'tcp';
    this.addr = addr; // [host, port] or 'host:port'
    this.worker = worker;
    this.conn = null;
    this.req = new Request();
    this.lastTime = getMillisecond();
    this.recvBuffer = Buffer.alloc(0);
    this._closed = false;

    // FIFO of resolvers waiting for a PDT_OK confirmation from the server.
    // Populated by expectOk(), drained by onOk().
    this._okResolvers = [];
  }

  // Wait for the next PDT_OK from the server. The server sends PDT_OK in
  // response to PDT_W_SET_NAME and PDT_W_ADD_FUNC, so awaiting this after each
  // registration packet guarantees the server has actually processed it before
  // we proceed (closes the "have no job do" race for early client calls).
  // A safety timeout prevents hanging forever if the OK is ever lost.
  expectOk(timeoutMs = 5000) {
    return new Promise((resolve) => {
      let done = false;
      const finish = () => {
        if (done) return;
        done = true;
        resolve();
      };
      this._okResolvers.push(finish);
      setTimeout(() => {
        if (!done) {
          const idx = this._okResolvers.indexOf(finish);
          if (idx !== -1) this._okResolvers.splice(idx, 1);
          if (this.worker) this.worker.emit('warning', new Error('PDT_OK timeout, proceeding anyway'));
          finish();
        }
      }, timeoutMs);
    });
  }

  onOk() {
    const finish = this._okResolvers.shift();
    if (finish) finish();
  }

  _parseAddr(addr) {
    if (Array.isArray(addr)) {
      return { host: addr[0], port: addr[1] };
    }
    if (typeof addr === 'string') {
      const idx = addr.lastIndexOf(':');
      if (idx === -1) throw new Error('invalid addr: ' + addr);
      return { host: addr.slice(0, idx), port: parseInt(addr.slice(idx + 1), 10) };
    }
    throw new Error('invalid addr: ' + addr);
  }

  // Connect to the server. Resolves on success, rejects on failure.
  connect() {
    return new Promise((resolve, reject) => {
      const { host, port } = this._parseAddr(this.addr);
      const socket = net.createConnection({ host, port });
      this.conn = socket;
      this._closed = false;

      let settled = false;
      socket.setTimeout(C.DIAL_TIME_OUT * 1000);
      socket.once('connect', () => {
        socket.setTimeout(0);
        settled = true;
        this.lastTime = getMillisecond();
        this.emit('connect');
        resolve();
      });
      socket.once('timeout', () => {
        if (!settled) {
          settled = true;
          socket.destroy(new Error('connect timeout'));
          reject(new Error('connect timeout'));
        }
      });
      socket.once('error', (err) => {
        if (!settled) {
          settled = true;
          reject(err);
        }
      });

      socket.on('data', (chunk) => this._onData(chunk));
      socket.on('error', (err) => this._onError(err));
      socket.on('close', () => this._onClose());
    });
  }

  // Reconnect an existing agent after a connection drop.
  reConnect() {
    return new Promise((resolve, reject) => {
      if (this.conn) {
        try { this.conn.destroy(); } catch (e) { /* ignore */ }
        this.conn = null;
      }
      const { host, port } = this._parseAddr(this.addr);
      const socket = net.createConnection({ host, port });
      this.conn = socket;
      this._closed = false;

      socket.setTimeout(C.DIAL_TIME_OUT * 1000);
      socket.once('connect', () => {
        socket.setTimeout(0);
        this.lastTime = getMillisecond();
        resolve();
      });
      socket.once('timeout', () => {
        socket.destroy(new Error('connect timeout'));
        reject(new Error('connect timeout'));
      });
      socket.once('error', reject);

      socket.on('data', (chunk) => this._onData(chunk));
      socket.on('error', (err) => this._onError(err));
      socket.on('close', () => this._onClose());
    });
  }

  _onData(chunk) {
    this.recvBuffer = this.recvBuffer.length
      ? Buffer.concat([this.recvBuffer, chunk])
      : chunk;
    this._processBuffer();
  }

  _processBuffer() {
    while (true) {
      if (this.recvBuffer.length < C.MIN_DATA_SIZE) return;
      const dataLen = this.recvBuffer.readUInt32BE(8);
      const packetLen = C.MIN_DATA_SIZE + dataLen;
      if (this.recvBuffer.length < packetLen) return;

      const packet = this.recvBuffer.subarray(0, packetLen);
      this.recvBuffer = this.recvBuffer.subarray(packetLen);

      const { resp, err } = decodePack(packet);
      if (err || !resp) continue;

      resp.agent = this;
      // Dispatch directly on the worker (single event loop, no queue needed).
      this.worker.handleResp(resp);
    }
  }

  write() {
    if (!this.conn || this._closed) return;
    const buf = this.req.encodePack();
    this.conn.write(buf);
  }

  heartBeatPing() {
    this.req.heartBeatPack();
    this.write();
  }

  grab() {
    this.req.grabDataPack();
    this.write();
  }

  wakeup() {
    this.req.wakeupPack();
    this.write();
  }

  limitExceed() {
    this.req.limitExceedPack();
    this.write();
  }

  delOldFuncMsg(funcName) {
    this.req.delFunctionPack(funcName);
    this.write();
  }

  reAddFuncMsg(funcName) {
    this.req.addFunctionPack(funcName);
    this.write();
  }

  reSetWorkerName(workerName) {
    this.req.setWorkerName(workerName);
    this.write();
  }

  _onError(err) {
    this.emit('error', err);
  }

  _onClose() {
    this.conn = null;
    this.emit('close');
  }

  close() {
    this._closed = true;
    if (this.conn) {
      try { this.conn.destroy(); } catch (e) { /* ignore */ }
      this.conn = null;
    }
  }
}

module.exports = Agent;
