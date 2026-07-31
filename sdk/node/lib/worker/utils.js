'use strict';

// Worker utility helpers. Mirrors sdk/python3/nmidsdk/worker/utils.py.

const { encode: msgpackEncode, decode: msgpackDecode } = require('@msgpack/msgpack');

let _counter = 0;

// Simple unique id generator for worker id/name, similar in spirit to the
// python IdGenerator (timestamp-based, monotonically increasing).
class IdGenerator {
  get_id() {
    _counter += 1;
    return `${Date.now()}-${_counter}`;
  }
}

function getMillisecond() {
  return Date.now();
}

function getNowSecond() {
  return Math.floor(Date.now() / 1000);
}

function msgpackParamsMap(params) {
  if (!params || params.length === 0) return null;
  return msgpackDecode(params);
}

function jsonParamsMap(params) {
  if (!params || params.length === 0) return null;
  return JSON.parse(params.toString('utf8'));
}

// Encode a value to a msgpack Buffer.
function msgpackPack(value) {
  return Buffer.from(msgpackEncode(value));
}

// Standard nmid return struct: { Code, Msg, Data }. Matches the Go
// wor.GetRetStruct() / python example ret_struct pattern.
function buildRetStruct(code, msg, data) {
  const obj = {
    Code: code,
    Msg: msg,
    Data: data == null ? Buffer.alloc(0) : data,
  };
  return msgpackPack(obj);
}

module.exports = {
  IdGenerator,
  getMillisecond,
  getNowSecond,
  msgpackParamsMap,
  jsonParamsMap,
  msgpackPack,
  buildRetStruct,
};
