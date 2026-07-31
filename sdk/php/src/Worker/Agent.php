<?php
namespace Nmid\Worker;

use Nmid\Constants;

// Worker agent: a single connection from a worker to one nmid server.
// Mirrors node lib/worker/agent.js.
//
// PHP is synchronous, so reads use stream_select. expectOk() blocks for the
// next PDT_OK during registration; the main loop calls onData() which drains
// whatever is currently readable.
class Agent
{
    private string $host;
    private int $port;
    public Worker $worker;

    /** @var resource|null */
    public $conn = null;
    public Request $req;
    public int $lastTime;
    public string $recvBuffer = '';
    private bool $closed = false;

    public function __construct(string $network, string $addr, Worker $worker)
    {
        // addr: "host:port"
        $idx = strrpos($addr, ':');
        if ($idx === false) {
            throw new \InvalidArgumentException("invalid addr: {$addr}");
        }
        $this->host = substr($addr, 0, $idx);
        $this->port = (int)substr($addr, $idx + 1);
        $this->worker = $worker;
        $this->req = new Request();
        $this->lastTime = (int)(microtime(true) * 1000);
    }

    public function connect(): void
    {
        $remote = "tcp://{$this->host}:{$this->port}";
        $errno = 0;
        $errstr = '';
        $this->conn = @stream_socket_client(
            $remote,
            $errno,
            $errstr,
            Constants::DIAL_TIME_OUT,
            STREAM_CLIENT_CONNECT
        );
        if ($this->conn === false) {
            throw new \RuntimeException("connect failed: {$errstr} ({$errno})");
        }
        stream_set_blocking($this->conn, false);
        $this->lastTime = (int)(microtime(true) * 1000);
        $this->recvBuffer = '';
        $this->closed = false;
    }

    // Block until one complete packet is available. $deadline is a microtime
    // (float) deadline; null means no timeout. Returns null on timeout/EOF.
    public function readOnePacket(?float $deadline = null): ?Response
    {
        while (true) {
            // Try to pull a complete packet from the buffer first.
            if (strlen($this->recvBuffer) >= Constants::MIN_DATA_SIZE) {
                $dataLen = unpack('N', substr($this->recvBuffer, 8, 4))[1];
                $packetLen = Constants::MIN_DATA_SIZE + $dataLen;
                if (strlen($this->recvBuffer) >= $packetLen) {
                    $packet = substr($this->recvBuffer, 0, $packetLen);
                    $this->recvBuffer = substr($this->recvBuffer, $packetLen);
                    return Response::decodePack($packet);
                }
            }

            if ($this->conn === null || $this->closed) {
                return null;
            }

            $timeout = $deadline !== null ? max(0, $deadline - microtime(true)) : 1;
            if ($timeout <= 0) {
                return null;
            }
            $read = [$this->conn];
            $write = null;
            $except = null;
            $n = @stream_select($read, $write, $except, (int)ceil($timeout));
            if ($n === false || $n === 0) {
                return null;
            }
            $chunk = fread($this->conn, 8192);
            if ($chunk === '' || $chunk === false) {
                return null; // EOF
            }
            $this->recvBuffer .= $chunk;
            $this->lastTime = (int)(microtime(true) * 1000);
        }
    }

    // Wait for the server's PDT_OK confirmation after SET_NAME / ADD_FUNC.
    // Mirrors node agent.expectOk(). A 5s safety timeout prevents hanging.
    public function expectOk(int $timeoutMs = 5000): void
    {
        $deadline = microtime(true) + $timeoutMs / 1000;
        while (true) {
            $resp = $this->readOnePacket($deadline);
            if ($resp === null) {
                break; // timeout or EOF
            }
            if ($resp->dataType === Constants::PDT_OK) {
                return;
            }
            // Not an OK — dispatch to the worker (shouldn't normally happen
            // during registration, but be safe).
            $resp->agent = $this;
            $this->worker->handleResp($resp);
        }
        fwrite(STDERR, "warning: PDT_OK timeout, proceeding anyway\n");
    }

    // Non-blocking drain: read whatever is currently available and dispatch
    // complete packets. Returns false if the connection has been closed.
    public function onData(): bool
    {
        while (true) {
            $read = [$this->conn];
            $write = null;
            $except = null;
            $n = @stream_select($read, $write, $except, 0);
            if ($n === false) {
                return false;
            }
            if ($n === 0) {
                break; // nothing more right now
            }
            $chunk = fread($this->conn, 8192);
            if ($chunk === '' || $chunk === false) {
                return false; // EOF
            }
            $this->recvBuffer .= $chunk;
            $this->lastTime = (int)(microtime(true) * 1000);
        }

        $this->processBuffer();
        return true;
    }

    private function processBuffer(): void
    {
        while (strlen($this->recvBuffer) >= Constants::MIN_DATA_SIZE) {
            $dataLen = unpack('N', substr($this->recvBuffer, 8, 4))[1];
            $packetLen = Constants::MIN_DATA_SIZE + $dataLen;
            if (strlen($this->recvBuffer) < $packetLen) {
                break;
            }
            $packet = substr($this->recvBuffer, 0, $packetLen);
            $this->recvBuffer = substr($this->recvBuffer, $packetLen);

            $resp = Response::decodePack($packet);
            if ($resp !== null) {
                $resp->agent = $this;
                $this->worker->handleResp($resp);
            }
        }
    }

    public function write(): void
    {
        if ($this->conn === null || $this->closed) {
            return;
        }
        $buf = $this->req->encodePack();
        $total = strlen($buf);
        $written = 0;
        while ($written < $total) {
            $n = @fwrite($this->conn, substr($buf, $written));
            if ($n === false || $n === 0) {
                // Would block: wait briefly for writability.
                $read = null;
                $write = [$this->conn];
                $except = null;
                if (@stream_select($read, $write, $except, 1) === 0) {
                    $this->closed = true;
                    return;
                }
                continue;
            }
            $written += $n;
        }
    }

    public function heartBeatPing(): void
    {
        $this->req->heartBeatPack();
        $this->write();
    }

    public function wakeup(): void
    {
        $this->req->wakeupPack();
        $this->write();
    }

    public function close(): void
    {
        $this->closed = true;
        if ($this->conn !== null) {
            @fclose($this->conn);
            $this->conn = null;
        }
    }
}
