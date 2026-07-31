'use strict';

// nmid client.
// Mirrors sdk/python3/nmidsdk/client/client.py, but uses Node's native event
// loop instead of the python threading+asyncio bridge. TCP reads are handled
// by the socket 'data' event; a small framer reassembles complete packets
// before decoding and dispatching them.

const net = require('net');
const { EventEmitter } = require('events');
const Request = require('./request');
const { getConnType, decodePack } = require('./response');
const C = require('../const');

class Client extends EventEmitter {
  // network: 'tcp' (only tcp supported, kept for API parity with python)
  // addr: [host, port] (like the python tuple) or 'host:port' string
  constructor(network, addr) {
    super();
    this.network = network || 'tcp';
    this.addr = addr;
    this.conn = null;
    this.req = null;
    this.ioTimeOut = null;
    this.errHandler = null;

    // handle -> queue of pending entries { resolve, reject, isPromise }
    // Responses carry the func name as handle, so we dispatch FIFO per handle.
    this.pending = new Map();
    this.recvBuffer = Buffer.alloc(0);
    this._closed = false;
  }

  setIoTimeOut(t) {
    this.ioTimeOut = t;
    return this;
  }

  // snake_case alias for python parity
  set_io_time_out(t) {
    return this.setIoTimeOut(t);
  }

  setParamsHandle(hType) {
    if (
      hType !== C.PARAMS_HANDLE_TYPE_ENCODE &&
      hType !== C.PARAMS_HANDLE_TYPE_ORIGINAL
    ) {
      return this;
    }
    if (!this.req) this.req = new Request();
    this.req.paramsHandleType = hType;
    return this;
  }

  // snake_case alias
  set_params_handle(hType) {
    return this.setParamsHandle(hType);
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

  // Connect to the nmid server. Resolves with the client on success.
  start() {
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
        this.emit('connect');
        resolve(this);
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
        } else {
          this._onError(err);
        }
      });

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

  // Reassemble complete packets from the TCP byte stream.
  _processBuffer() {
    while (true) {
      if (this.recvBuffer.length < C.MIN_DATA_SIZE) return;
      const dataLen = this.recvBuffer.readUInt32BE(8);
      const packetLen = C.MIN_DATA_SIZE + dataLen;
      if (this.recvBuffer.length < packetLen) return;

      const packet = this.recvBuffer.subarray(0, packetLen);
      this.recvBuffer = this.recvBuffer.subarray(packetLen);

      if (getConnType(packet) !== C.CONN_TYPE_SERVER) {
        // ignore packets not coming from the server
        continue;
      }

      const { resp, err } = decodePack(packet);
      if (err || !resp) continue;
      this._processResp(resp);
    }
  }

  _processResp(resp) {
    const dt = resp.dataType;
    if (dt === C.PDT_ERROR || dt === C.PDT_CANT_DO || dt === C.PDT_RATELIMIT) {
      const e = resp.getResError();
      this._failAll(e);
      if (this.errHandler) this.errHandler(e);
      return;
    }
    if (dt === C.PDT_S_RETURN_DATA) {
      this._handleResp(resp);
    }
  }

  _handleResp(resp) {
    if (!resp.handle || resp.handleLen === 0) return;
    const queue = this.pending.get(resp.handle);
    if (queue && queue.length) {
      const entry = queue.shift();
      if (queue.length === 0) this.pending.delete(resp.handle);
      try {
        if (entry.resolve) entry.resolve(resp);
      } catch (e) {
        if (this.errHandler) this.errHandler(e);
      }
    }
  }

  _enqueue(handle, entry) {
    if (!this.pending.has(handle)) this.pending.set(handle, []);
    this.pending.get(handle).push(entry);
  }

  // Reject every outstanding promise entry (e.g. on connection loss / global
  // protocol error). Callback-style entries simply never resolve, matching
  // the python behavior where err_handler is the global sink.
  _failAll(err) {
    for (const queue of this.pending.values()) {
      for (const entry of queue) {
        if (entry.isPromise && entry.reject) entry.reject(err);
      }
    }
    this.pending.clear();
  }

  _send(funcName, params) {
    if (!this.req) this.req = new Request();
    this.req.contentPack(C.PDT_C_DO_JOB, funcName, params);
    const buf = this.req.encodePack();
    this.conn.write(buf);
  }

  // Callback style, matching the python API:
  //   client.do('ToUpper', paramsBuf, (resp) => { ... })
  do(funcName, params, callback) {
    if (!this.conn || this._closed) {
      throw new Error('Connection is not established');
    }
    this._enqueue(funcName, { resolve: callback, reject: null, isPromise: false });
    this._send(funcName, params);
  }

  // Promise style convenience:
  //   const resp = await client.doAsync('ToUpper', paramsBuf)
  doAsync(funcName, params) {
    if (!this.conn || this._closed) {
      return Promise.reject(new Error('Connection is not established'));
    }
    return new Promise((resolve, reject) => {
      this._enqueue(funcName, { resolve, reject, isPromise: true });
      this._send(funcName, params);
    });
  }

  _onError(err) {
    if (this.errHandler) this.errHandler(err);
    this._failAll(err);
    this.emit('error', err);
  }

  _onClose() {
    this.conn = null;
    this._failAll(new Error('connection closed'));
    this.emit('close');
  }

  close() {
    this._closed = true;
    if (this.conn) {
      this.conn.destroy();
      this.conn = null;
    }
    this._failAll(new Error('client closed'));
  }
}

module.exports = Client;
