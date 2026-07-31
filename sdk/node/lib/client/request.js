'use strict';

// Client request encoder.
// Mirrors sdk/python3/nmidsdk/client/request.py and pkg/client/request.go.
//
// Packet layout (big-endian):
//   header (12 bytes): ConnType(4) | DataType(4) | DataLen(4)
//   body (DataLen bytes) for PDT_C_DO_JOB:
//     ParamsType(4) | ParamsHandleType(4) | HandleLen(4) | Handle |
//     ParamsLen(4) | Params

const C = require('../const');

class Request {
  constructor() {
    this.dataType = 0;
    this.data = Buffer.alloc(0);
    this.dataLen = 0;

    this.handle = '';
    this.handleLen = 0;
    this.paramsType = C.PARAMS_TYPE_MSGPACK;
    this.paramsHandleType = C.PARAMS_HANDLE_TYPE_ENCODE;
    this.paramsLen = 0;
    this.params = Buffer.alloc(0);

    this.ret = Buffer.alloc(0);
    this.retLen = 0;
  }

  // Build the body for a client do-job request.
  contentPack(dataType, handle, params) {
    this.dataType = dataType;
    this.handle = handle;
    this.handleLen = Buffer.byteLength(handle, 'utf8');
    this.params = params;
    this.paramsLen = params.length;
    this.dataLen =
      C.UINT32_SIZE + // paramsType
      C.UINT32_SIZE + // paramsHandleType
      C.UINT32_SIZE + // handleLen
      this.handleLen +
      C.UINT32_SIZE + // paramsLen
      this.paramsLen;

    const content = Buffer.alloc(this.dataLen);
    let offset = 0;
    content.writeUInt32BE(this.paramsType, offset);
    offset += C.UINT32_SIZE;
    content.writeUInt32BE(this.paramsHandleType, offset);
    offset += C.UINT32_SIZE;
    content.writeUInt32BE(this.handleLen, offset);
    offset += C.UINT32_SIZE;
    content.write(handle, offset, this.handleLen, 'utf8');
    offset += this.handleLen;
    content.writeUInt32BE(this.paramsLen, offset);
    offset += C.UINT32_SIZE;
    this.params.copy(content, offset);

    this.data = content;
    return content;
  }

  // Prepend the 12-byte header.
  encodePack() {
    const total = C.MIN_DATA_SIZE + this.dataLen;
    const data = Buffer.alloc(total);
    data.writeUInt32BE(C.CONN_TYPE_CLIENT, 0);
    data.writeUInt32BE(this.dataType, C.UINT32_SIZE);
    data.writeUInt32BE(this.dataLen, C.UINT32_SIZE * 2);
    this.data.copy(data, C.MIN_DATA_SIZE);
    return data;
  }
}

module.exports = Request;
