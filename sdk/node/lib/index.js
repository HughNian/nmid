'use strict';

// nmid node.js SDK entry point.
//
// Usage:
//   const { Client, Worker, Const, buildRetStruct, msgpackPack } = require('nmidsdk');

const Client = require('./client/client');
const Worker = require('./worker/worker');
const Const = require('./const');

const { Response: ClientResponse } = require('./client/response');
const { Response: WorkerResponse } = require('./worker/response');
const {
  buildRetStruct,
  msgpackPack,
  msgpackParamsMap,
  jsonParamsMap,
  IdGenerator,
} = require('./worker/utils');

module.exports = {
  Client,
  Worker,
  Const,
  ClientResponse,
  WorkerResponse,
  buildRetStruct,
  msgpackPack,
  msgpackParamsMap,
  jsonParamsMap,
  IdGenerator,
};
