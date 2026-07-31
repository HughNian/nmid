'use strict';

// Client response decoder.
// Mirrors sdk/python3/nmidsdk/client/response.py and pkg/client/response.go.
//
// For PDT_S_RETURN_DATA the body (read from the packet at MIN_DATA_SIZE) is:
//   HandleLen(4) | ParamsLen(4) | RetLen(4) |
//   Handle | Params | Ret

const C = require('../const');

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

    this.ret = Buffer.alloc(0);
    this.retLen = 0;
  }

  getResError() {
    let err;
    switch (this.dataType) {
      case C.PDT_ERROR:
        err = new Error('request error');
        err.code = 'PDT_ERROR';
        break;
      case C.PDT_CANT_DO:
        err = new Error('have no job do');
        err.code = 'PDT_CANT_DO';
        break;
      case C.PDT_RATELIMIT:
        err = new Error('have ratelimit');
        err.code = 'PDT_RATELIMIT';
        break;
      default:
        return null;
    }
    return err;
  }

  getResResult() {
    if (this.dataType === C.PDT_S_RETURN_DATA) {
      return this.ret;
    }
    return null;
  }
}

// Read the conn type from the first 4 bytes of a packet.
function getConnType(data) {
  if (!data || data.length < C.UINT32_SIZE) {
    return 0;
  }
  return data.readUInt32BE(0);
}

// Decode a single complete packet (>= MIN_DATA_SIZE + dataLen bytes) into a
// Response. Returns { resp, err }.
function decodePack(data) {
  const resLen = data.length;
  if (resLen < C.MIN_DATA_SIZE) {
    return { resp: null, err: new Error('Invalid data1: too short') };
  }

  const cl = data.readUInt32BE(8); // content length
  if (resLen < C.MIN_DATA_SIZE + cl) {
    return { resp: null, err: new Error('Invalid data2: incomplete body') };
  }

  const content = data.subarray(C.MIN_DATA_SIZE, C.MIN_DATA_SIZE + cl);
  if (content.length !== cl) {
    return { resp: null, err: new Error('Invalid data3: content length mismatch') };
  }

  const resp = new Response();
  resp.dataType = data.readUInt32BE(4);
  resp.dataLen = cl;
  resp.data = content;

  if (resp.dataType === C.PDT_S_RETURN_DATA) {
    let start = C.MIN_DATA_SIZE;
    resp.handleLen = data.readUInt32BE(start);
    start += C.UINT32_SIZE;
    resp.paramsLen = data.readUInt32BE(start);
    start += C.UINT32_SIZE;
    resp.retLen = data.readUInt32BE(start);
    start += C.UINT32_SIZE;
    resp.handle = data.toString('utf8', start, start + resp.handleLen);
    start += resp.handleLen;
    resp.params = data.subarray(start, start + resp.paramsLen);
    start += resp.paramsLen;
    resp.ret = data.subarray(start, start + resp.retLen);
  }

  return { resp, err: null };
}

module.exports = { Response, getConnType, decodePack };
