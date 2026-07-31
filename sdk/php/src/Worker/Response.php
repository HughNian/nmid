<?php
namespace Nmid\Worker;

use Nmid\Constants;
use Nmid\Msgpack;

// Worker response decoder for PDT_S_GET_DATA.
// Mirrors node lib/worker/response.js.
//
// body (PDT_S_GET_DATA):
//   ParamsType(4) | ParamsHandleType(4) | HandleLen(4) | ParamsLen(4) |
//   JobIdLen(4) | Handle | Params | JobId
class Response
{
    public int $dataType = 0;
    public string $data = '';
    public int $dataLen = 0;

    public string $handle = '';
    public int $handleLen = 0;

    public int $paramsType = 0;
    public int $paramsHandleType = 0;
    public int $paramsLen = 0;
    public string $params = '';

    /** @var array|null */
    public $paramsMap = null;

    public string $jobId = '';
    public int $jobIdLen = 0;

    /** @var Agent|null back-reference to the agent that received this job */
    public $agent = null;

    public function getParamsMap(): ?array
    {
        return $this->paramsLen === 0 ? null : $this->paramsMap;
    }

    private function parseParams(string $params): void
    {
        $this->params = $params;
        if ($this->paramsType === Constants::PARAMS_TYPE_MSGPACK) {
            $this->paramsMap = Msgpack::decode($params);
        } elseif ($this->paramsType === Constants::PARAMS_TYPE_JSON) {
            $this->paramsMap = json_decode($params, true);
        }
    }

    public static function decodePack(string $packet): ?Response
    {
        $len = strlen($packet);
        if ($len < Constants::MIN_DATA_SIZE) {
            return null;
        }

        $header = unpack('N3', substr($packet, 0, Constants::MIN_DATA_SIZE));
        $dataType = $header[2];
        $dataLen = $header[3];

        if ($len < Constants::MIN_DATA_SIZE + $dataLen) {
            return null;
        }

        $resp = new self();
        $resp->dataType = $dataType;
        $resp->dataLen = $dataLen;
        $resp->data = substr($packet, Constants::MIN_DATA_SIZE, $dataLen);

        if ($dataType === Constants::PDT_S_GET_DATA) {
            $p = Constants::MIN_DATA_SIZE;
            $resp->paramsType = unpack('N', substr($packet, $p, 4))[1];
            $p += 4;
            $resp->paramsHandleType = unpack('N', substr($packet, $p, 4))[1];
            $p += 4;
            $resp->handleLen = unpack('N', substr($packet, $p, 4))[1];
            $p += 4;
            $resp->paramsLen = unpack('N', substr($packet, $p, 4))[1];
            $p += 4;
            $resp->jobIdLen = unpack('N', substr($packet, $p, 4))[1];
            $p += 4;

            $resp->handle = substr($packet, $p, $resp->handleLen);
            $p += $resp->handleLen;

            $params = substr($packet, $p, $resp->paramsLen);
            $resp->parseParams($params);
            $p += $resp->paramsLen;

            $resp->jobId = substr($packet, $p, $resp->jobIdLen);
        }

        return $resp;
    }
}
