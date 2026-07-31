'use strict';

// Worker response decoder.
// Mirrors sdk/python3/nmidsdk/worker/response.py and pkg/worker/response.go.
//
// For PDT_S_GET_DATA the body (read from the packet at MIN_DATA_SIZE) is:
//   ParamsType(4) | ParamsHandleType(4) | HandleLen(4) | ParamsLen(4) |
//   JobIdLen(4) | Handle | Params | JobId
// Params are parsed (msgpack or json) into paramsMap.

const C = require('../const');
const { msgpackParamsMap, jsonParamsMap } = require('./utils');

class Response {
  constructor() {
    this.dataType = 0;
    this.data = Buffer.alloc(0);
    this.dataLen = 0;

    this.handle = '';
    this.handleLen = 0;
    this.paramsType = 0;
    this.paramsHandleType = 0;
    this.paramsLen = 0;
    this.params = Buffer.alloc(0);
    this.paramsMap = null;
    this.jobId = '';
    this.jobIdLen = 0;

    this.ret = Buffer.alloc(0);
    this.retLen = 0;

    this.agent = null; // back-reference to the Agent that received this job
  }

  getResponse() {
    return this;
  }

  getParams() {
    if (this.paramsLen === 0) return null;
    return this.params;
  }

  getParamsMap() {
    if (this.paramsLen === 0) return null;
    return this.paramsMap;
  }

  // Merge paramsMap into the given object, mirroring python should_bind.
  shouldBind(obj) {
    if (this.paramsMap) Object.assign(obj, this.paramsMap);
  }

  parseParams(params) {
    this.params = params;
    if (this.paramsType === C.PARAMS_TYPE_MSGPACK) {
      this.paramsMap = msgpackParamsMap(params);
    } else if (this.paramsType === C.PARAMS_TYPE_JSON) {
      this.paramsMap = jsonParamsMap(params);
    }
  }
}

// Decode a single complete packet (>= MIN_DATA_SIZE + dataLen) into a
// Response. Returns { resp, err }.
function decodePack(data) {
  const resLen = data.length;
  if (resLen < C.MIN_DATA_SIZE) {
    return { resp: null, err: new Error('Invalid data: too short') };
  }

  const cl = data.readUInt32BE(8); // content length
  if (resLen < C.MIN_DATA_SIZE + cl) {
    return { resp: null, err: new Error('Invalid data: incomplete body') };
  }

  const content = data.subarray(C.MIN_DATA_SIZE, C.MIN_DATA_SIZE + cl);
  if (content.length !== cl) {
    return { resp: null, err: new Error('Invalid data: content length mismatch') };
  }

  const resp = new Response();
  resp.dataType = data.readUInt32BE(4);
  resp.dataLen = cl;
  resp.data = content;

  if (resp.dataType === C.PDT_S_GET_DATA) {
    let start = C.MIN_DATA_SIZE;
    resp.paramsType = data.readUInt32BE(start);
    start += C.UINT32_SIZE;
    resp.paramsHandleType = data.readUInt32BE(start);
    start += C.UINT32_SIZE;
    resp.handleLen = data.readUInt32BE(start);
    start += C.UINT32_SIZE;
    resp.paramsLen = data.readUInt32BE(start);
    start += C.UINT32_SIZE;
    resp.jobIdLen = data.readUInt32BE(start);
    start += C.UINT32_SIZE;

    resp.handle = data.toString('utf8', start, start + resp.handleLen);
    start += resp.handleLen;

    resp.parseParams(data.subarray(start, start + resp.paramsLen));
    start += resp.paramsLen;

    resp.jobId = data.toString('utf8', start, start + resp.jobIdLen);
  }

  return { resp, err: null };
}

module.exports = { Response, decodePack };
