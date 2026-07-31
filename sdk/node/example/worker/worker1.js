'use strict';

// nmid worker example.
// Mirrors sdk/python3/example/worker/worker1/worker.py.
//
// Run a nmid server (127.0.0.1:6808) first, then:
//   node example/worker/worker1.js

const { Worker, buildRetStruct } = require('../../lib');

const NMID_SERVER_HOST = '127.0.0.1';
const NMID_SERVER_PORT = 6808;

// jobFunc receives the decoded WorkerResponse and must return a Buffer (the
// ret payload, typically a msgpack-encoded RetStruct).
async function toUpper(job) {
  const resp = job.getResponse();
  if (!resp) {
    throw new Error('response data error');
  }

  const paramsMap = resp.getParamsMap();
  if (!paramsMap || !paramsMap.name) {
    throw new Error('params error');
  }

  const name = String(paramsMap.name);
  const data = Buffer.from(name.toUpperCase(), 'utf8');
  return buildRetStruct(0, 'ok', data);
}

async function main() {
  const worker = new Worker();
  worker.setWorkerName('Worker1');

  worker.addServer('tcp', [NMID_SERVER_HOST, NMID_SERVER_PORT]);
  worker.addFunction('ToUpper', toUpper);

  worker.on('error', (err) => {
    console.error('worker error:', err && err.message ? err.message : err);
  });

  const err = await worker.workerReady();
  if (err) {
    console.error('worker ready error:', err.message);
    worker.workerClose();
    return;
  }

  await worker.workerDo();
  console.log('worker started, funcs:ToUpper');

  // graceful shutdown on SIGINT/SIGTERM
  const shutdown = () => {
    worker.workerClose();
    process.exit(0);
  };
  process.on('SIGINT', shutdown);
  process.on('SIGTERM', shutdown);
}

main();
