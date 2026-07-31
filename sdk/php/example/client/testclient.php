<?php
// nmid client example.
// Mirrors node example/client/testclient.js.
//
// Run: php example/client/testclient.php
// (requires the nmid server on 127.0.0.1:6808 and a worker exposing "ToUpper")

require __DIR__ . '/../../bootstrap.php';

use Nmid\Client\Client;
use Nmid\Client\NmidError;
use Nmid\Msgpack;
use Nmid\RetStruct;

const SERVER_HOST = '127.0.0.1';
const SERVER_PORT = 6808;

// msgpack-encode {name: "nihaonihao"}
function encodeParams(string $name): string
{
    return Msgpack::encode(['name' => $name]);
}

// PDT_CANT_DO is transient (worker may still be registering or reconnecting):
// retry a few times before giving up.
function callWithRetry(Client $client, string $funcName, string $params, int $retries = 5, int $delayMs = 300): ?string
{
    for ($i = 0; $i < $retries; $i++) {
        try {
            return $client->doJob($funcName, $params);
        } catch (NmidError $e) {
            if ($e->errorCode === 'PDT_CANT_DO' && $i < $retries - 1) {
                usleep($delayMs * 1000);
                continue;
            }
            fwrite(STDERR, "Client error: " . $e->getMessage() . "\n");
            return null;
        } catch (\Throwable $e) {
            fwrite(STDERR, "Client error: " . $e->getMessage() . "\n");
            return null;
        }
    }
    fwrite(STDERR, "call failed after {$retries} retries\n");
    return null;
}

$addr = SERVER_HOST . ':' . SERVER_PORT;
$client = new Client($addr);

try {
    $client->start();
} catch (\Throwable $e) {
    fwrite(STDERR, "Error starting client: " . $e->getMessage() . "\n");
    exit(1);
}

$params = encodeParams('nihaonihao');
$ret = callWithRetry($client, 'ToUpper', $params);

if ($ret !== null && $ret !== '') {
    $parsed = RetStruct::parse($ret);
    if ($parsed === null) {
        fwrite(STDERR, "failed to decode ret\n");
    } else {
        [$code, $msg, $data] = $parsed;
        if ($code !== 0) {
            echo $msg . "\n";
        } else {
            echo $data . "\n";
        }
    }
}

$client->close();
