'use strict';

// nmid client example.
// Mirrors sdk/python3/example/client/testclient/client.py.
//
// Run a nmid server (127.0.0.1:6808) and a worker exposing "ToUpper" first,
// then:
//   node example/client/testclient.js

const { Client, Const } = require('../../lib');
const { encode: msgpackEncode, decode: msgpackDecode } = require('@msgpack/msgpack');

const SERVER_HOST = '127.0.0.1';
const SERVER_PORT = 6808;

function errHandler(e) {
  if (e && e.code === 'PDT_CANT_DO') {
    // transient: no worker registered for the function right now (worker may
    // still be starting up, or momentarily reconnecting)
    console.warn('have no job do (transient, will retry)');
  } else {
    console.error('Client error:', e && e.message ? e.message : e);
  }
}

function handleResponse(resp) {
  if (!resp || resp.dataType !== Const.PDT_S_RETURN_DATA || resp.retLen === 0) {
    return null;
  }
  const retStruct = msgpackDecode(resp.ret);
  if (retStruct.Code !== 0) {
    console.log(retStruct.Msg);
    return null;
  }
  const out = Buffer.from(retStruct.Data).toString('utf8');
  console.log(out);
  return out;
}

// doAsync rejects on PDT_CANT_DO when no worker is available yet. That is a
// transient condition (worker still registering, or mid-reconnect), so retry
// a few times before giving up.
async function callWithRetry(client, funcName, params, retries = 5, delayMs = 300) {
  let lastErr;
  for (let i = 0; i < retries; i++) {
    try {
      return await client.doAsync(funcName, params);
    } catch (e) {
      lastErr = e;
      if (e && e.code === 'PDT_CANT_DO' && i < retries - 1) {
        await new Promise((r) => setTimeout(r, delayMs));
        continue;
      }
      throw e;
    }
  }
  throw lastErr;
}

async function main() {
  const client = new Client('tcp', [SERVER_HOST, SERVER_PORT]);
  client.errHandler = errHandler;

  try {
    await client.start();
  } catch (e) {
    console.error('Error starting client:', e.message);
    return;
  }

  const params = Buffer.from(msgpackEncode({ name: 'nihaonihao' }));

  try {
    const resp = await callWithRetry(client, 'ToUpper', params);
    handleResponse(resp);
  } catch (e) {
    errHandler(e);
  } finally {
    client.close();
  }
}

main();
