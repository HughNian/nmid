<?php
namespace Nmid\Client;

use Nmid\Constants;

// Client request encoder for PDT_C_DO_JOB.
// Mirrors node lib/client/request.js.
//
// header (big-endian): ConnType(4)=CONN_TYPE_CLIENT | DataType(4) | DataLen(4)
// body (PDT_C_DO_JOB):
//   ParamsType(4) | ParamsHandleType(4) | HandleLen(4) | Handle |
//   ParamsLen(4) | Params
class Request
{
    public int $dataType = 0;
    public string $data = '';
    public int $dataLen = 0;

    public string $handle = '';
    public int $handleLen = 0;

    public int $paramsType = Constants::PARAMS_TYPE_MSGPACK;
    public int $paramsHandleType = Constants::PARAMS_HANDLE_TYPE_ENCODE;
    public int $paramsLen = 0;
    public string $params = '';

    public function contentPack(int $dataType, string $handle, string $params): void
    {
        $this->dataType = $dataType;
        $this->handle = $handle;
        $this->handleLen = strlen($handle);
        $this->params = $params;
        $this->paramsLen = strlen($params);

        $this->dataLen = Constants::UINT32_SIZE   // paramsType
            + Constants::UINT32_SIZE              // paramsHandleType
            + Constants::UINT32_SIZE              // handleLen
            + $this->handleLen
            + Constants::UINT32_SIZE              // paramsLen
            + $this->paramsLen;

        $body = pack(
            'N3',
            $this->paramsType,
            $this->paramsHandleType,
            $this->handleLen
        ) . $this->handle . pack('N', $this->paramsLen) . $this->params;

        $this->data = $body;
    }

    public function encodePack(): string
    {
        $header = pack(
            'N3',
            Constants::CONN_TYPE_CLIENT,
            $this->dataType,
            $this->dataLen
        );
        return $header . $this->data;
    }
}
