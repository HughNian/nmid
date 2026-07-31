'use strict';

// Worker request encoder.
// Mirrors sdk/python3/nmidsdk/worker/request.py and pkg/worker/request.go.
//
// Every worker packet starts with header (big-endian):
//   ConnType(4)=CONN_TYPE_WORKER | DataType(4) | DataLen(4)
// followed by DataLen bytes of body. Most control packets carry the body as
// plain utf-8 bytes (function name / worker name / "PING"); the return-data
// packet (PDT_W_RETURN_DATA) carries a structured body (see retPack).

const C = require('../const');

class Request {
  constructor() {
    this.dataType = 0;
    this.data = Buffer.alloc(0);
    this.dataLen = 0;

    this.handle = '';
    this.handleLen = 0;
    this.paramsType = 0;
    this.paramsLen = 0;
    this.params = Buffer.alloc(0);
    this.jobId = '';
    this.jobIdLen = 0;
    this.ret = Buffer.alloc(0);
    this.retLen = 0;
  }

  _setBytes(dataType, payload) {
    this.dataType = dataType;
    this.data = payload;
    this.dataLen = payload.length;
    return this.data;
  }

  heartBeatPack() {
    return this._setBytes(C.PDT_W_HEARTBEAT_PING, Buffer.from('PING', 'utf8'));
  }

  setWorkerName(workerName) {
    return this._setBytes(C.PDT_W_SET_NAME, Buffer.from(workerName, 'utf8'));
  }

  addFunctionPack(funcName) {
    return this._setBytes(C.PDT_W_ADD_FUNC, Buffer.from(funcName, 'utf8'));
  }

  delFunctionPack(funcName) {
    return this._setBytes(C.PDT_W_DEL_FUNC, Buffer.from(funcName, 'utf8'));
  }

  grabDataPack() {
    return this._setBytes(C.PDT_W_GRAB_JOB, Buffer.alloc(0));
  }

  wakeupPack() {
    return this._setBytes(C.PDT_WAKEUP, Buffer.alloc(0));
  }

  limitExceedPack() {
    return this._setBytes(C.PDT_RATELIMIT, Buffer.alloc(0));
  }

  // Build the PDT_W_RETURN_DATA body, echoing back the original job's handle,
  // params and jobId together with the computed ret.
  //
  // body layout:
  //   HandleLen(4) | Handle | ParamsLen(4) | Params |
  //   RetLen(4) | Ret | JobIdLen(4) | JobId
  retPack(ret) {
    this.ret = ret;
    this.retLen = ret.length;

    this.dataType = C.PDT_W_RETURN_DATA;
    this.dataLen =
      C.UINT32_SIZE + this.handleLen +
      C.UINT32_SIZE + this.paramsLen +
      C.UINT32_SIZE + this.retLen +
      C.UINT32_SIZE + this.jobIdLen;

    const content = Buffer.alloc(this.dataLen);
    let offset = 0;

    content.writeUInt32BE(this.handleLen, offset);
    offset += C.UINT32_SIZE;
    content.write(this.handle, offset, this.handleLen, 'utf8');
    offset += this.handleLen;

    content.writeUInt32BE(this.paramsLen, offset);
    offset += C.UINT32_SIZE;
    this.params.copy(content, offset);
    offset += this.paramsLen;

    content.writeUInt32BE(this.retLen, offset);
    offset += C.UINT32_SIZE;
    this.ret.copy(content, offset);
    offset += this.retLen;

    content.writeUInt32BE(this.jobIdLen, offset);
    offset += C.UINT32_SIZE;
    content.write(this.jobId, offset, this.jobIdLen, 'utf8');

    this.data = content;
    return content;
  }

  // Prepend the 12-byte header (CONN_TYPE_WORKER).
  encodePack() {
    const total = C.MIN_DATA_SIZE + this.dataLen;
    const data = Buffer.alloc(total);
    data.writeUInt32BE(C.CONN_TYPE_WORKER, 0);
    data.writeUInt32BE(this.dataType, C.UINT32_SIZE);
    data.writeUInt32BE(this.dataLen, C.UINT32_SIZE * 2);
    this.data.copy(data, C.MIN_DATA_SIZE);
    return data;
  }
}

module.exports = Request;
