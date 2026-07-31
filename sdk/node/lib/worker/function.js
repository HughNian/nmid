'use strict';

// A registered worker function. Mirrors sdk/python3/nmidsdk/worker/function.py.
//
// jobFunc signature: (resp: WorkerResponse) => Buffer | Promise<Buffer>
// The function receives the decoded job response and must return the ret
// payload (typically a msgpack-encoded RetStruct). It may throw to signal
// failure; returning a Promise is supported.

class Function {
  constructor(jobFunc, fname) {
    this.func = jobFunc;
    this.funcName = fname;
  }
}

module.exports = Function;
