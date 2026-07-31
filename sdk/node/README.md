## nmidsdk (node.js)

`nmidsdk` is a Node.js SDK for the [nmid](https://github.com/HughNian/nmid) micro
service RPC framework. It implements the same wire protocol and feature set as
the python3 SDK (`sdk/python3`), providing both a **client** and a **worker**.

Like the other nmid SDKs, the wire protocol uses a 12-byte big-endian header
(`ConnType | DataType | DataLen`) followed by the body, with
[msgpack](https://msgpack.org/) as the default payload format.

### Installation

```bash
cd sdk/node
npm install
```

Then in your project (if linked locally):

```js
const { Client, Worker, Const, buildRetStruct } = require('nmidsdk');
```

### Constants

All protocol constants live in `lib/const.js` and are re-exported as `Const`.
The most used ones:

| constant | meaning |
| --- | --- |
| `PDT_C_DO_JOB` (16) | client -> server: do a job |
| `PDT_S_RETURN_DATA` (11) | server -> client: job result |
| `PDT_S_GET_DATA` (10) | server -> worker: a job to run |
| `PDT_W_RETURN_DATA` (15) | worker -> server: job result |
| `PDT_W_ADD_FUNC` (13) / `PDT_W_DEL_FUNC` (14) | worker register/unregister a function |
| `PDT_W_SET_NAME` (23) | worker set its name |
| `PDT_W_HEARTBEAT_PING` (22) / `PDT_S_HEARTBEAT_PONG` (23) | heartbeat |
| `PARAMS_TYPE_MSGPACK` (5) / `PARAMS_TYPE_JSON` (6) | params encoding |
| `CONN_TYPE_CLIENT` (3) / `CONN_TYPE_WORKER` (2) / `CONN_TYPE_SERVER` (1) | conn type |

### Client

```js
const { Client, Const } = require('nmidsdk');
const { encode: msgpackEncode, decode: msgpackDecode } = require('@msgpack/msgpack');

const client = new Client('tcp', ['127.0.0.1', 6808]); // or '127.0.0.1:6808'
client.errHandler = (err) => console.error('client error:', err.message);

await client.start();

const params = Buffer.from(msgpackEncode({ name: 'nihaonihao' }));

// callback style (matches the python API)
client.do('ToUpper', params, (resp) => {
  if (resp.dataType === Const.PDT_S_RETURN_DATA && resp.retLen !== 0) {
    const ret = msgpackDecode(resp.ret);
    console.log(Buffer.from(ret.Data).toString('utf8'));
  }
  client.close();
});

// promise style alternative
// const resp = await client.doAsync('ToUpper', params);
// ...
// client.close();
```

API:

- `new Client(network, addr)` — `addr` is `[host, port]` or `'host:port'`.
- `client.start()` → `Promise<client>` — connect to the nmid server.
- `client.errHandler = fn` — global error sink (protocol errors, socket errors).
- `client.setIoTimeOut(ms)` — set an io timeout.
- `client.setParamsHandle(type)` — `PARAMS_HANDLE_TYPE_ENCODE` (default) or `_ORIGINAL`.
- `client.do(funcName, paramsBuf, callback)` — send a job, invoke `callback(resp)` on the matching `PDT_S_RETURN_DATA`.
- `client.doAsync(funcName, paramsBuf)` → `Promise<resp>` — promise variant.
- `client.close()` — close the connection.

### Worker

```js
const { Worker, buildRetStruct } = require('nmidsdk');

async function toUpper(job) {
  const resp = job.getResponse();
  const paramsMap = resp.getParamsMap(); // { name: '...' }
  const data = Buffer.from(String(paramsMap.name).toUpperCase(), 'utf8');
  return buildRetStruct(0, 'ok', data); // { Code, Msg, Data } msgpack buffer
}

const worker = new Worker();
worker.setWorkerName('Worker1');
worker.addServer('tcp', ['127.0.0.1', 6808]);
worker.addFunction('ToUpper', toUpper);

worker.on('error', (err) => console.error('worker error:', err.message));

const err = await worker.workerReady();
if (err) { worker.workerClose(); return; }

await worker.workerDo();
```

API:

- `new Worker()`
- `worker.setWorkerName(name)` / `worker.setWorkerId(id)` — auto-generated when omitted.
- `worker.addServer(network, addr)` — add an nmid server connection (an internal `Agent`).
- `worker.addFunction(funcName, jobFunc)` / `worker.delFunction(funcName)` — register/unregister a function.
  - `jobFunc(job)` receives the worker response (`job.getResponse()`, `job.getParamsMap()`). It returns a `Buffer` (the ret payload) or a `Promise<Buffer>`, and may throw.
- `worker.workerReady()` → `Promise<err|null>` — connect all agents and register functions.
- `worker.workerDo()` → `Promise` — start the heartbeat and process incoming jobs.
- `worker.workerClose()` — deregister functions and close all connections.

### Return struct

`buildRetStruct(code, msg, data)` produces the standard nmid return payload —
a msgpack-encoded `{ Code, Msg, Data }` map — matching `wor.GetRetStruct()` in
the go worker and the `ret_struct` pattern in the python examples.

### Examples

```bash
# terminal 1: nmid server
cd cmd/server && go run main.go

# terminal 2: worker
node sdk/node/example/worker/worker1.js

# terminal 3: client
node sdk/node/example/client/testclient.js
```

### Tests

A protocol round-trip self-test (no server needed) verifies the binary layout
against the python/go SDKs:

```bash
cd sdk/node
npm test
```
