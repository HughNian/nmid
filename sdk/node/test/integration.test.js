'use strict';

// End-to-end integration test.
// A fake nmid "server" relays between the real Client and the real Worker over
// real TCP sockets, so the full socket read/write/framing/dispatch path is
// exercised without needing the go nmid binary.

const net = require('net');
const assert = require('assert');
const { encode: msgpackEncode, decode: msgpackDecode } = require('@msgpack/msgpack');

const { Client, Worker, Const, buildRetStruct } = require('../lib');

const HOST = '127.0.0.1';
let PORT = 0; // assigned by server.listen

// ---- minimal nmid packet helpers used by the fake server -----------------

function readPacket(buf) {
  // returns [packetBuffer, remaining] or null if not enough data yet
  if (buf.length < Const.MIN_DATA_SIZE) return null;
  const dataLen = buf.readUInt32BE(8);
  const packetLen = Const.MIN_DATA_SIZE + dataLen;
  if (buf.length < packetLen) return null;
  return [buf.subarray(0, packetLen), buf.subarray(packetLen)];
}

// Parse a client PDT_C_DO_JOB body, returns { handle, params }
function parseDoJobBody(body) {
  let o = 0;
  /* paramsType */ o += Const.UINT32_SIZE;
  /* paramsHandleType */ o += Const.UINT32_SIZE;
  const handleLen = body.readUInt32BE(o); o += Const.UINT32_SIZE;
  const handle = body.toString('utf8', o, o + handleLen); o += handleLen;
  const paramsLen = body.readUInt32BE(o); o += Const.UINT32_SIZE;
  const params = body.subarray(o, o + paramsLen);
  return { handle, params };
}

// Parse a worker PDT_W_RETURN_DATA body, returns { handle, ret, jobId }
function parseReturnBody(body) {
  let o = 0;
  const handleLen = body.readUInt32BE(o); o += Const.UINT32_SIZE;
  const handle = body.toString('utf8', o, o + handleLen); o += handleLen;
  const paramsLen = body.readUInt32BE(o); o += Const.UINT32_SIZE;
  o += paramsLen; // skip echoed params
  const retLen = body.readUInt32BE(o); o += Const.UINT32_SIZE;
  const ret = body.subarray(o, o + retLen); o += retLen;
  const jobIdLen = body.readUInt32BE(o); o += Const.UINT32_SIZE;
  const jobId = body.toString('utf8', o, o + jobIdLen);
  return { handle, ret, jobId };
}

// Build a PDT_S_GET_DATA packet (server -> worker) for a job.
function buildGetData(handle, params, jobId) {
  const handleB = Buffer.from(handle, 'utf8');
  const jobIdB = Buffer.from(jobId, 'utf8');
  const bodyLen =
    Const.UINT32_SIZE * 5 + handleB.length + params.length + jobIdB.length;
  const pkt = Buffer.alloc(Const.MIN_DATA_SIZE + bodyLen);
  pkt.writeUInt32BE(Const.CONN_TYPE_SERVER, 0);
  pkt.writeUInt32BE(Const.PDT_S_GET_DATA, 4);
  pkt.writeUInt32BE(bodyLen, 8);
  let o = Const.MIN_DATA_SIZE;
  pkt.writeUInt32BE(Const.PARAMS_TYPE_MSGPACK, o); o += Const.UINT32_SIZE;
  pkt.writeUInt32BE(Const.PARAMS_HANDLE_TYPE_ENCODE, o); o += Const.UINT32_SIZE;
  pkt.writeUInt32BE(handleB.length, o); o += Const.UINT32_SIZE;
  pkt.writeUInt32BE(params.length, o); o += Const.UINT32_SIZE;
  pkt.writeUInt32BE(jobIdB.length, o); o += Const.UINT32_SIZE;
  handleB.copy(pkt, o); o += handleB.length;
  params.copy(pkt, o); o += params.length;
  jobIdB.copy(pkt, o);
  return pkt;
}

// Build a PDT_S_RETURN_DATA packet (server -> client).
function buildReturnData(handle, ret) {
  const handleB = Buffer.from(handle, 'utf8');
  const bodyLen = Const.UINT32_SIZE * 3 + handleB.length + ret.length;
  const pkt = Buffer.alloc(Const.MIN_DATA_SIZE + bodyLen);
  pkt.writeUInt32BE(Const.CONN_TYPE_SERVER, 0);
  pkt.writeUInt32BE(Const.PDT_S_RETURN_DATA, 4);
  pkt.writeUInt32BE(bodyLen, 8);
  let o = Const.MIN_DATA_SIZE;
  pkt.writeUInt32BE(handleB.length, o); o += Const.UINT32_SIZE;
  pkt.writeUInt32BE(0, o); o += Const.UINT32_SIZE; // paramsLen 0
  pkt.writeUInt32BE(ret.length, o); o += Const.UINT32_SIZE;
  handleB.copy(pkt, o); o += handleB.length;
  ret.copy(pkt, o);
  return pkt;
}

// Build a header-only (empty body) server packet, e.g. PDT_OK / PONG.
function buildEmpty(dataType) {
  const pkt = Buffer.alloc(Const.MIN_DATA_SIZE);
  pkt.writeUInt32BE(Const.CONN_TYPE_SERVER, 0);
  pkt.writeUInt32BE(dataType, 4);
  pkt.writeUInt32BE(0, 8); // dataLen 0
  return pkt;
}

// ---- fake server ----------------------------------------------------------

function startFakeServer() {
  return new Promise((resolve) => {
    const workerConns = []; // sockets that registered at least one function
    const clientJobs = []; // { socket, handle, params }
    const bufBySocket = new Map();

    const server = net.createServer((sock) => {
      bufBySocket.set(sock, Buffer.alloc(0));
      sock.on('data', (chunk) => {
        let buf = Buffer.concat([bufBySocket.get(sock), chunk]);
        bufBySocket.set(sock, buf);
        while (true) {
          const r = readPacket(bufBySocket.get(sock));
          if (!r) break;
          const pkt = r[0];
          bufBySocket.set(sock, r[1]);

          const connType = pkt.readUInt32BE(0);
          const dataType = pkt.readUInt32BE(4);
          const body = pkt.subarray(Const.MIN_DATA_SIZE);

          if (connType === Const.CONN_TYPE_WORKER) {
            if (dataType === Const.PDT_W_SET_NAME) {
              // real server responds with PDT_OK
              sock.write(buildEmpty(Const.PDT_OK));
            }
            if (dataType === Const.PDT_W_ADD_FUNC) {
              if (!workerConns.includes(sock)) workerConns.push(sock);
              // real server responds with PDT_OK after registering the function
              sock.write(buildEmpty(Const.PDT_OK));
            }
            if (dataType === Const.PDT_W_HEARTBEAT_PING) {
              sock.write(buildEmpty(Const.PDT_S_HEARTBEAT_PONG));
            }
            if (dataType === Const.PDT_W_RETURN_DATA) {
              const { handle, ret, jobId } = parseReturnBody(body);
              // find matching pending client job by handle, send result back
              const idx = clientJobs.findIndex((j) => j.handle === handle);
              if (idx !== -1) {
                const job = clientJobs.splice(idx, 1)[0];
                job.socket.write(buildReturnData(handle, ret));
              }
            }
          } else if (connType === Const.CONN_TYPE_CLIENT) {
            if (dataType === Const.PDT_C_DO_JOB) {
              const { handle, params } = parseDoJobBody(body);
              clientJobs.push({ socket: sock, handle, params });
              // dispatch to the first available worker
              const worker = workerConns[0];
              if (worker) {
                const job = clientJobs[clientJobs.length - 1];
                worker.write(buildGetData(job.handle, job.params, 'job-1'));
              }
            }
          }
        }
      });
      sock.on('close', () => bufBySocket.delete(sock));
      sock.on('error', () => {});
    });

    server.listen(0, HOST, () => {
      PORT = server.address().port;
      resolve(server);
    });
  });
}

// ---- the test -------------------------------------------------------------

async function main() {
  console.log('integration test (client + worker over fake server):');
  const server = await startFakeServer();
  const addr = [HOST, PORT];

  // worker
  const worker = new Worker();
  worker.setWorkerName('Worker1');
  worker.addServer('tcp', addr);
  worker.addFunction('ToUpper', async (job) => {
    const m = job.getParamsMap();
    return buildRetStruct(0, 'ok', Buffer.from(String(m.name).toUpperCase(), 'utf8'));
  });
  worker.on('error', (e) => { throw e; });
  const rerr = await worker.workerReady();
  assert.ifError(rerr);
  await worker.workerDo();

  // give the server a tick to register the worker's ADD_FUNC
  await new Promise((r) => setTimeout(r, 100));

  // client
  const client = new Client('tcp', addr);
  client.errHandler = (e) => { throw e; };
  await client.start();

  const params = Buffer.from(msgpackEncode({ name: 'hello world' }));

  const resp = await client.doAsync('ToUpper', params);
  assert.strictEqual(resp.dataType, Const.PDT_S_RETURN_DATA);
  assert.strictEqual(resp.handle, 'ToUpper');
  const ret = msgpackDecode(resp.ret);
  assert.strictEqual(ret.Code, 0);
  assert.strictEqual(Buffer.from(ret.Data).toString('utf8'), 'HELLO WORLD');

  console.log('  ok - client.doAsync(ToUpper) -> HELLO WORLD');

  // callback style too
  const params2 = Buffer.from(msgpackEncode({ name: 'nmid' }));
  await new Promise((resolve, reject) => {
    client.do('ToUpper', params2, (r2) => {
      try {
        const rt = msgpackDecode(r2.ret);
        assert.strictEqual(Buffer.from(rt.Data).toString('utf8'), 'NMID');
        resolve();
      } catch (e) { reject(e); }
    });
  });
  console.log('  ok - client.do(ToUpper, cb) -> NMID');

  client.close();
  worker.workerClose();
  server.close();

  console.log('\nintegration test passed');
}

main().catch((e) => {
  console.error('integration test failed:', e);
  process.exit(1);
});
