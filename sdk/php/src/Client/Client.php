<?php
namespace Nmid\Client;

use Nmid\Constants;

// nmid client. Mirrors node lib/client/client.js.
//
// PHP is synchronous/blocking, so doJob() blocks until the response arrives
// (or an error packet is received). TCP is done with stream_socket_client +
// fread/fwrite; packet reassembly uses a recv buffer.
class Client
{
    private string $host;
    private int $port;

    /** @var resource|null */
    private $conn = null;
    private string $recvBuffer = '';
    private bool $closed = false;

    public function __construct(string $addr)
    {
        // accept "host:port"
        $idx = strrpos($addr, ':');
        if ($idx === false) {
            throw new \InvalidArgumentException("invalid addr: {$addr}");
        }
        $this->host = substr($addr, 0, $idx);
        $this->port = (int)substr($addr, $idx + 1);
    }

    public function start(): void
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
        stream_set_blocking($this->conn, true);
        $this->closed = false;
    }

    // Synchronous call: send PDT_C_DO_JOB, wait for the matching response.
    // On PDT_CANT_DO/PDT_ERROR/PDT_RATELIMIT throws NmidError so the caller
    // can retry transient failures.
    public function doJob(string $funcName, string $params): string
    {
        if ($this->conn === null || $this->closed) {
            throw new \RuntimeException('connection is not established');
        }

        $req = new Request();
        $req->contentPack(Constants::PDT_C_DO_JOB, $funcName, $params);
        $buf = $req->encodePack();

        $written = fwrite($this->conn, $buf);
        if ($written === false || $written !== strlen($buf)) {
            throw new \RuntimeException('write failed');
        }

        // Read until we get a complete packet destined for us.
        while (true) {
            $packet = $this->readPacket();
            if ($packet === null) {
                throw new \RuntimeException('connection closed by server');
            }

            // Only handle server packets.
            $connType = unpack('N', substr($packet, 0, 4))[1];
            if ($connType !== Constants::CONN_TYPE_SERVER) {
                continue;
            }

            $resp = Response::decodePack($packet);
            if ($resp === null) {
                continue;
            }

            // Error packets are global (server can't satisfy the request).
            $err = NmidError::fromDataType($resp->dataType);
            if ($err !== null) {
                throw $err;
            }

            if ($resp->dataType === Constants::PDT_S_RETURN_DATA) {
                return $resp->ret;
            }

            // Ignore anything else (e.g. heartbeat) — shouldn't happen for client.
        }
    }

    // Read one complete packet from the socket, blocking until enough bytes
    // are available. Returns null on connection close.
    private function readPacket(): ?string
    {
        while (true) {
            if (strlen($this->recvBuffer) >= Constants::MIN_DATA_SIZE) {
                $dataLen = unpack('N', substr($this->recvBuffer, 8, 4))[1];
                $packetLen = Constants::MIN_DATA_SIZE + $dataLen;
                if (strlen($this->recvBuffer) >= $packetLen) {
                    $packet = substr($this->recvBuffer, 0, $packetLen);
                    $this->recvBuffer = substr($this->recvBuffer, $packetLen);
                    return $packet;
                }
            }

            $chunk = fread($this->conn, 8192);
            if ($chunk === '' || $chunk === false) {
                return null; // EOF / closed
            }
            $this->recvBuffer .= $chunk;
        }
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
