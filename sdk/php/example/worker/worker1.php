<?php
// nmid worker example.
// Mirrors node example/worker/worker1.js.
//
// Run: php example/worker/worker1.php
// (requires the nmid server on 127.0.0.1:6808)

require __DIR__ . '/../../bootstrap.php';

use Nmid\Worker\Worker;
use Nmid\Worker\Response;

const SERVER_HOST = '127.0.0.1';
const SERVER_PORT = 6808;

$wor = new Worker();
$wor->setWorkerId();
$wor->setWorkerName();
$wor->addServer('tcp', SERVER_HOST . ':' . SERVER_PORT);

// ToUpper: upper-case the "name" field of the params.
$wor->addFunction('ToUpper', function (Response $resp): string {
    $paramsMap = $resp->getParamsMap();
    if (empty($paramsMap['name'])) {
        throw new RuntimeException('params error');
    }
    $name = (string)$paramsMap['name'];
    $data = strtoupper($name);
    return Worker::buildRet(0, 'ok', $data);
});

fwrite(STDERR, "worker started, funcs:ToUpper\n");

// graceful shutdown on Ctrl+C
pcntl_async_signals(true);
pcntl_signal(SIGINT, function () use ($wor) {
    fwrite(STDERR, "Shutting down...\n");
    $wor->workerClose();
    exit(0);
});

try {
    $wor->workerDo();
} catch (\Throwable $e) {
    fwrite(STDERR, "error: " . $e->getMessage() . "\n");
    exit(1);
}
