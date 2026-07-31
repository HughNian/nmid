<?php
namespace Nmid\Worker;

use Nmid\Constants;

// Worker request encoder.
// Mirrors node lib/worker/request.js.
//
// header (big-endian): ConnType(4)=CONN_TYPE_WORKER | DataType(4) | DataLen(4)
// Most control packets carry plain utf-8 bytes as the body (worker name,
// function name, "PING"). PDT_W_RETURN_DATA carries a structured body.
class Request
{
    public int $dataType = 0;
    public string $data = '';
    public int $dataLen = 0;

    public string $handle = '';
    public int $handleLen = 0;
    public string $params = '';
    public int $paramsLen = 0;
    public string $jobId = '';
    public int $jobIdLen = 0;
    public string $ret = '';
    public int $retLen = 0;

    private function setBytes(int $dataType, string $payload): void
    {
        $this->dataType = $dataType;
        $this->data = $payload;
        $this->dataLen = strlen($payload);
    }

    public function heartBeatPack(): void
    {
        $this->setBytes(Constants::PDT_W_HEARTBEAT_PING, 'PING');
    }

    public function setWorkerName(string $name): void
    {
        $this->setBytes(Constants::PDT_W_SET_NAME, $name);
    }

    public function addFunctionPack(string $funcName): void
    {
        $this->setBytes(Constants::PDT_W_ADD_FUNC, $funcName);
    }

    public function delFunctionPack(string $funcName): void
    {
        $this->setBytes(Constants::PDT_W_DEL_FUNC, $funcName);
    }

    public function wakeupPack(): void
    {
        $this->setBytes(Constants::PDT_WAKEUP, '');
    }

    // Build the PDT_W_RETURN_DATA body, echoing back the original job's
    // handle / params / jobId together with the computed ret.
    // body: HandleLen(4)|Handle|ParamsLen(4)|Params|RetLen(4)|Ret|JobIdLen(4)|JobId
    public function returnDataPack(Response $resp, string $ret): void
    {
        $this->handle = $resp->handle;
        $this->handleLen = $resp->handleLen;
        $this->params = $resp->params;
        $this->paramsLen = $resp->paramsLen;
        $this->jobId = $resp->jobId;
        $this->jobIdLen = $resp->jobIdLen;
        $this->ret = $ret;
        $this->retLen = strlen($ret);

        $this->dataType = Constants::PDT_W_RETURN_DATA;
        $this->dataLen = Constants::UINT32_SIZE + $this->handleLen
            + Constants::UINT32_SIZE + $this->paramsLen
            + Constants::UINT32_SIZE + $this->retLen
            + Constants::UINT32_SIZE + $this->jobIdLen;

        $this->data = pack('N', $this->handleLen) . $this->handle
            . pack('N', $this->paramsLen) . $this->params
            . pack('N', $this->retLen) . $this->ret
            . pack('N', $this->jobIdLen) . $this->jobId;
    }

    public function encodePack(): string
    {
        return pack('N3', Constants::CONN_TYPE_WORKER, $this->dataType, $this->dataLen)
            . $this->data;
    }
}
