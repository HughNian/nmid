<?php
namespace Nmid\Worker;

use Nmid\Constants;
use Nmid\RetStruct;

// nmid worker. Mirrors node lib/worker/worker.js.
//
// Holds one or more Agents (connections to nmid servers) and a set of
// registered functions. The main loop (workerDo) uses stream_select to wait
// for jobs, dispatches them to the registered callable, and returns the ret.
// A heartbeat ping is sent on a fixed interval; a stale agent (no PONG within
// NMID_SERVER_TIMEOUT) is reconnected and re-registered.
class Worker
{
    public string $workerId = '';
    public string $workerName = '';
    /** @var Agent[] */
    public array $agents = [];
    /** @var callable[] name => callable(Response):string */
    public array $funcs = [];
    public int $funcsNum = 0;
    public bool $ready = false;
    public bool $running = false;

    public function setWorkerId(string $wid = ''): self
    {
        $this->workerId = $wid !== '' ? $wid : $this->genId();
        return $this;
    }

    public function setWorkerName(string $wname = ''): self
    {
        $this->workerName = $wname !== '' ? $wname : $this->genId();
        return $this;
    }

    private function genId(): string
    {
        return substr((string)(microtime(true) * 1000), 0, 10) . '-' . mt_rand();
    }

    // network: 'tcp'; addr: "host:port"
    public function addServer(string $network, string $addr): void
    {
        $this->agents[] = new Agent($network, $addr, $this);
    }

    public function addFunction(string $funcName, callable $jobFunc): void
    {
        if (isset($this->funcs[$funcName])) {
            throw new \RuntimeException("function {$funcName} already exist");
        }
        $this->funcs[$funcName] = $jobFunc;
        $this->funcsNum++;
    }

    public function delFunction(string $funcName): void
    {
        if (!isset($this->funcs[$funcName])) {
            throw new \RuntimeException("function {$funcName} not exist");
        }
        unset($this->funcs[$funcName]);
        $this->funcsNum--;
    }

    // Connect all agents, set worker name and register all functions, waiting
    // for PDT_OK after each registration packet (avoids the "have no job do"
    // race for early client calls).
    public function workerReady(): ?\Throwable
    {
        if (empty($this->agents)) {
            return new \RuntimeException('none active agents');
        }
        if ($this->funcsNum === 0 || empty($this->funcs)) {
            return new \RuntimeException('none funcs');
        }

        if ($this->workerName === '') {
            $this->setWorkerName();
        }

        foreach ($this->agents as $agent) {
            try {
                $agent->connect();
            } catch (\Throwable $e) {
                return $e;
            }

            $agent->req->setWorkerName($this->workerName);
            $agent->write();
            $agent->expectOk();

            foreach (array_keys($this->funcs) as $fname) {
                $agent->req->addFunctionPack($fname);
                $agent->write();
                $agent->expectOk();
            }
        }

        $this->ready = true;
        return null;
    }

    // Dispatch an incoming decoded response to the right handler.
    public function handleResp(Response $resp): void
    {
        $dt = $resp->dataType;

        if ($dt === Constants::PDT_OK) {
            // Handled by expectOk() during registration.
            return;
        }

        if ($dt === Constants::PDT_TOSLEEP) {
            // Server asks us to sleep briefly then wake up. PHP is synchronous,
            // so just send wakeup immediately.
            if ($resp->agent !== null) {
                $resp->agent->wakeup();
            }
            return;
        }

        if ($dt === Constants::PDT_S_GET_DATA) {
            $err = $this->doFunction($resp);
            if ($err !== null) {
                fwrite(STDERR, "error: " . $err->getMessage() . "\n");
            }
            return;
        }

        if ($dt === Constants::PDT_S_HEARTBEAT_PONG) {
            if ($resp->agent !== null) {
                $resp->agent->lastTime = (int)(microtime(true) * 1000);
            }
            return;
        }

        // PDT_NO_JOB / PDT_WAKEUPED: nothing to do.
    }

    // Execute the registered function for a job and return its ret to the server.
    public function doFunction(Response $resp): ?\Throwable
    {
        if (!isset($this->funcs[$resp->handle])) {
            return new \RuntimeException("function {$resp->handle} not found");
        }
        if ($resp->paramsLen === 0) {
            return new \RuntimeException('params error');
        }

        $agent = $resp->agent;
        try {
            $ret = ($this->funcs[$resp->handle])($resp);
            if (!is_string($ret)) {
                return new \RuntimeException('job function must return a string');
            }
            $agent->req->returnDataPack($resp, $ret);
            $agent->write();
            return null;
        } catch (\Throwable $e) {
            return $e;
        }
    }

    public function heartBeat(): void
    {
        foreach ($this->agents as $agent) {
            if ($agent->conn !== null) {
                $agent->heartBeatPing();
            }
        }
    }

    public function workerReConnect(Agent $agent): void
    {
        $agent->close();
        fwrite(STDERR, "warning: reconnecting agent\n");
        try {
            $agent->connect();
        } catch (\Throwable $e) {
            fwrite(STDERR, "reconnect failed: " . $e->getMessage() . "\n");
            return;
        }

        // Re-register and wait for PDT_OK, same as workerReady.
        $agent->req->setWorkerName($this->workerName);
        $agent->write();
        $agent->expectOk();
        foreach (array_keys($this->funcs) as $fname) {
            $agent->req->addFunctionPack($fname);
            $agent->write();
            $agent->expectOk();
        }
    }

    // Main loop: connect + register, then run heartbeat + timeout + job loop.
    public function workerDo(): void
    {
        if (!$this->ready) {
            $e = $this->workerReady();
            if ($e !== null) {
                throw $e;
            }
        }

        $this->running = true;
        $this->heartBeat();
        $lastHeartbeat = microtime(true);

        while ($this->running) {
            $read = [];
            $agentByStream = [];
            foreach ($this->agents as $agent) {
                if ($agent->conn !== null) {
                    $read[] = $agent->conn;
                    $agentByStream[(int)$agent->conn] = $agent;
                }
            }
            if (empty($read)) {
                break;
            }

            $write = null;
            $except = null;
            // 1s select so we can still fire heartbeats / timeout checks.
            $n = @stream_select($read, $write, $except, 1);
            $now = microtime(true);

            if ($n === false) {
                // Interrupted (e.g. by a signal); loop again.
                continue;
            }

            if ($n > 0) {
                foreach ($read as $stream) {
                    $agent = $agentByStream[(int)$stream] ?? null;
                    if ($agent === null) {
                        continue;
                    }
                    if (!$agent->onData()) {
                        // Connection closed — reconnect.
                        $this->workerReConnect($agent);
                    }
                }
            }

            // Heartbeat on a fixed interval.
            if ($now - $lastHeartbeat >= Constants::DEFAULT_HEARTBEAT_TIME) {
                $this->heartBeat();
                $lastHeartbeat = $now;
            }

            // Reconnect stale agents (no PONG within NMID_SERVER_TIMEOUT).
            foreach ($this->agents as $agent) {
                if ($agent->conn !== null
                    && ($now * 1000 - $agent->lastTime) > Constants::NMID_SERVER_TIMEOUT) {
                    $this->workerReConnect($agent);
                }
            }
        }
    }

    public function workerClose(): void
    {
        $this->running = false;
        foreach ($this->agents as $agent) {
            $agent->close();
        }
    }

    // Helper for example code to build a standard RetStruct.
    public static function buildRet(int $code, string $msg, string $data = ''): string
    {
        return RetStruct::build($code, $msg, $data);
    }
}
