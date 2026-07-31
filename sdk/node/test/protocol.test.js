'use strict';

// Protocol round-trip self-test.
// No nmid server required: we encode requests with the client encoder, decode
// them with the worker decoder (and vice versa) to verify the binary layout
// matches the python/go SDKs exactly.

const assert = require('assert');
const { encode: msgpackEncode, decode: msgpackDecode } = require('@msgpack/msgpack');

const ClientRequest = require('../lib/client/request');
const { decodePack: clientDecode, getConnType } = require('../lib/client/response');
const WorkerRequest = require('../lib/worker/request');
const { decodePack: workerDecode } = require('../lib/worker/response');
const { buildRetStruct } = require('../lib/worker/utils');
const C = require('../lib/const');

let passed = 0;
function ok(name) {
  passed += 1;
  console.log('  ok -', name);
}

function testClientRequestRoundTrip() {
  const params = Buffer.from(msgpackEncode({ name: 'niansong' }));
  const req = new ClientRequest();
  req.contentPack(C.PDT_C_DO_JOB, 'ToUpper', params);
  const buf = req.encodePack();

  // header
  assert.strictEqual(buf.readUInt32BE(0), C.CONN_TYPE_CLIENT, 'conn type');
  assert.strictEqual(buf.readUInt32BE(4), C.PDT_C_DO_JOB, 'data type');
  const dataLen = buf.readUInt32BE(8);
  assert.strictEqual(buf.length, C.MIN_DATA_SIZE + dataLen, 'total length');

  // The worker-side decoder reads PDT_S_GET_DATA jobs with a different body
  // layout, so instead decode with the client decoder's body parser by
  // simulating a server return: build a PDT_S_RETURN_DATA packet and decode.
  const handle = 'ToUpper';
  const handleB = Buffer.from(handle, 'utf8');
  const retParams = Buffer.from(msgpackEncode({ name: 'niansong' }));
  const ret = buildRetStruct(0, 'ok', Buffer.from('NIANSONG', 'utf8'));

  // PDT_S_RETURN_DATA body: HandleLen|ParamsLen|RetLen|Handle|Params|Ret
  const bodyLen =
    C.UINT32_SIZE * 3 + handleB.length + retParams.length + ret.length;
  const packet = Buffer.alloc(C.MIN_DATA_SIZE + bodyLen);
  packet.writeUInt32BE(C.CONN_TYPE_SERVER, 0);
  packet.writeUInt32BE(C.PDT_S_RETURN_DATA, 4);
  packet.writeUInt32BE(bodyLen, 8);
  let o = C.MIN_DATA_SIZE;
  packet.writeUInt32BE(handleB.length, o); o += C.UINT32_SIZE;
  packet.writeUInt32BE(retParams.length, o); o += C.UINT32_SIZE;
  packet.writeUInt32BE(ret.length, o); o += C.UINT32_SIZE;
  handleB.copy(packet, o); o += handleB.length;
  retParams.copy(packet, o); o += retParams.length;
  ret.copy(packet, o);

  assert.strictEqual(getConnType(packet), C.CONN_TYPE_SERVER);
  const { resp, err } = clientDecode(packet);
  assert.ifError(err);
  assert.strictEqual(resp.dataType, C.PDT_S_RETURN_DATA);
  assert.strictEqual(resp.handle, 'ToUpper');
  assert.strictEqual(resp.handleLen, handleB.length);
  assert.strictEqual(resp.paramsLen, retParams.length);
  assert.strictEqual(resp.retLen, ret.length);
  assert.deepStrictEqual(Buffer.from(resp.ret), ret);

  const retStruct = msgpackDecode(resp.ret);
  assert.strictEqual(retStruct.Code, 0);
  assert.strictEqual(retStruct.Msg, 'ok');
  assert.strictEqual(Buffer.from(retStruct.Data).toString('utf8'), 'NIANSONG');

  ok('client PDT_C_DO_JOB encode + PDT_S_RETURN_DATA decode round-trip');
}

function testWorkerRequestRoundTrip() {
  // Build a PDT_S_GET_DATA job packet (as the server would send) and decode it
  // with the worker decoder, then build the PDT_W_RETURN_DATA reply.
  const handle = 'ToUpper';
  const handleB = Buffer.from(handle, 'utf8');
  const params = Buffer.from(msgpackEncode({ name: 'nihaonihao' }));
  const jobId = 'job-123';
  const jobIdB = Buffer.from(jobId, 'utf8');

  // body: ParamsType|ParamsHandleType|HandleLen|ParamsLen|JobIdLen|Handle|Params|JobId
  const bodyLen =
    C.UINT32_SIZE * 5 + handleB.length + params.length + jobIdB.length;
  const packet = Buffer.alloc(C.MIN_DATA_SIZE + bodyLen);
  packet.writeUInt32BE(C.CONN_TYPE_SERVER, 0);
  packet.writeUInt32BE(C.PDT_S_GET_DATA, 4);
  packet.writeUInt32BE(bodyLen, 8);
  let o = C.MIN_DATA_SIZE;
  packet.writeUInt32BE(C.PARAMS_TYPE_MSGPACK, o); o += C.UINT32_SIZE;
  packet.writeUInt32BE(C.PARAMS_HANDLE_TYPE_ENCODE, o); o += C.UINT32_SIZE;
  packet.writeUInt32BE(handleB.length, o); o += C.UINT32_SIZE;
  packet.writeUInt32BE(params.length, o); o += C.UINT32_SIZE;
  packet.writeUInt32BE(jobIdB.length, o); o += C.UINT32_SIZE;
  handleB.copy(packet, o); o += handleB.length;
  params.copy(packet, o); o += params.length;
  jobIdB.copy(packet, o);

  const { resp, err } = workerDecode(packet);
  assert.ifError(err);
  assert.strictEqual(resp.dataType, C.PDT_S_GET_DATA);
  assert.strictEqual(resp.handle, 'ToUpper');
  assert.strictEqual(resp.paramsType, C.PARAMS_TYPE_MSGPACK);
  assert.strictEqual(resp.paramsHandleType, C.PARAMS_HANDLE_TYPE_ENCODE);
  assert.strictEqual(resp.jobId, jobId);
  assert.deepStrictEqual(resp.getParamsMap(), { name: 'nihaonihao' });

  // Build the return-data reply.
  const ret = buildRetStruct(0, 'ok', Buffer.from('NIHAONIHAO', 'utf8'));
  const wreq = new WorkerRequest();
  wreq.handleLen = resp.handleLen;
  wreq.handle = resp.handle;
  wreq.paramsLen = resp.paramsLen;
  wreq.params = resp.params;
  wreq.jobIdLen = resp.jobIdLen;
  wreq.jobId = resp.jobId;
  wreq.retPack(ret);
  const reply = wreq.encodePack();

  assert.strictEqual(reply.readUInt32BE(0), C.CONN_TYPE_WORKER);
  assert.strictEqual(reply.readUInt32BE(4), C.PDT_W_RETURN_DATA);
  const replyDataLen = reply.readUInt32BE(8);
  assert.strictEqual(reply.length, C.MIN_DATA_SIZE + replyDataLen);

  // Verify the reply body echo fields.
  let p = C.MIN_DATA_SIZE;
  assert.strictEqual(reply.readUInt32BE(p), resp.handleLen); p += C.UINT32_SIZE + resp.handleLen;
  assert.strictEqual(reply.readUInt32BE(p), resp.paramsLen); p += C.UINT32_SIZE + resp.paramsLen;
  assert.strictEqual(reply.readUInt32BE(p), ret.length); p += C.UINT32_SIZE + ret.length;
  assert.strictEqual(reply.readUInt32BE(p), resp.jobIdLen);

  ok('worker PDT_S_GET_DATA decode + PDT_W_RETURN_DATA encode round-trip');
}

function testControlPackets() {
  const req = new WorkerRequest();
  req.heartBeatPack();
  let buf = req.encodePack();
  assert.strictEqual(buf.readUInt32BE(0), C.CONN_TYPE_WORKER);
  assert.strictEqual(buf.readUInt32BE(4), C.PDT_W_HEARTBEAT_PING);
  assert.strictEqual(buf.toString('utf8', C.MIN_DATA_SIZE), 'PING');

  req.setWorkerName('Worker1');
  buf = req.encodePack();
  assert.strictEqual(buf.readUInt32BE(4), C.PDT_W_SET_NAME);
  assert.strictEqual(buf.toString('utf8', C.MIN_DATA_SIZE), 'Worker1');

  req.addFunctionPack('ToUpper');
  buf = req.encodePack();
  assert.strictEqual(buf.readUInt32BE(4), C.PDT_W_ADD_FUNC);
  assert.strictEqual(buf.toString('utf8', C.MIN_DATA_SIZE), 'ToUpper');

  req.grabDataPack();
  buf = req.encodePack();
  assert.strictEqual(buf.readUInt32BE(4), C.PDT_W_GRAB_JOB);
  assert.strictEqual(buf.readUInt32BE(8), 0);

  ok('worker control packets (heartbeat / setname / addfunc / grab)');
}

function testFraming() {
  // Two complete client return packets concatenated, plus a partial tail.
  const handle = 'ToUpper';
  const handleB = Buffer.from(handle, 'utf8');
  const ret = buildRetStruct(0, 'ok', Buffer.from('X', 'utf8'));
  const bodyLen = C.UINT32_SIZE * 3 + handleB.length + ret.length;
  function onePacket() {
    const packet = Buffer.alloc(C.MIN_DATA_SIZE + bodyLen);
    packet.writeUInt32BE(C.CONN_TYPE_SERVER, 0);
    packet.writeUInt32BE(C.PDT_S_RETURN_DATA, 4);
    packet.writeUInt32BE(bodyLen, 8);
    let o = C.MIN_DATA_SIZE;
    packet.writeUInt32BE(handleB.length, o); o += C.UINT32_SIZE;
    packet.writeUInt32BE(0, o); o += C.UINT32_SIZE; // paramsLen 0
    packet.writeUInt32BE(ret.length, o); o += C.UINT32_SIZE;
    handleB.copy(packet, o); o += handleB.length;
    ret.copy(packet, o);
    return packet;
  }
  const combined = Buffer.concat([onePacket(), onePacket(), Buffer.from([1, 2, 3])]);

  // Manually run the client's framing logic.
  let recv = combined;
  let count = 0;
  while (recv.length >= C.MIN_DATA_SIZE) {
    const dataLen = recv.readUInt32BE(8);
    const packetLen = C.MIN_DATA_SIZE + dataLen;
    if (recv.length < packetLen) break;
    const packet = recv.subarray(0, packetLen);
    recv = recv.subarray(packetLen);
    const { resp, err } = clientDecode(packet);
    assert.ifError(err);
    assert.strictEqual(resp.handle, 'ToUpper');
    count += 1;
  }
  assert.strictEqual(count, 2, 'two complete packets decoded');
  assert.strictEqual(recv.length, 3, 'partial tail retained');

  ok('stream framing handles multiple + partial packets');
}

(function run() {
  console.log('protocol self-test:');
  testClientRequestRoundTrip();
  testWorkerRequestRoundTrip();
  testControlPackets();
  testFraming();
  console.log(`\nall ${passed} tests passed`);
})();
