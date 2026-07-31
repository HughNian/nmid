<?php
namespace Nmid\Client;

use Nmid\Constants;

// Client response decoder for PDT_S_RETURN_DATA.
// Mirrors node lib/client/response.js.
//
// body (PDT_S_RETURN_DATA):
//   HandleLen(4) | ParamsLen(4) | RetLen(4) | Handle | Params | Ret
class Response
{
    public int $dataType = 0;
    public string $data = '';
    public int $dataLen = 0;

    public string $handle = '';
    public int $handleLen = 0;

    public int $paramsLen = 0;
    public string $params = '';

    public string $ret = '';
    public int $retLen = 0;

    // Decode one complete packet (>= MIN_DATA_SIZE + dataLen bytes).
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

        if ($dataType === Constants::PDT_S_RETURN_DATA) {
            $p = Constants::MIN_DATA_SIZE;
            $resp->handleLen = unpack('N', substr($packet, $p, 4))[1];
            $p += 4;
            $resp->paramsLen = unpack('N', substr($packet, $p, 4))[1];
            $p += 4;
            $resp->retLen = unpack('N', substr($packet, $p, 4))[1];
            $p += 4;
            $resp->handle = substr($packet, $p, $resp->handleLen);
            $p += $resp->handleLen;
            $resp->params = substr($packet, $p, $resp->paramsLen);
            $p += $resp->paramsLen;
            $resp->ret = substr($packet, $p, $resp->retLen);
        }

        return $resp;
    }

    public function getRet(): ?string
    {
        if ($this->dataType === Constants::PDT_S_RETURN_DATA) {
            return $this->ret;
        }
        return null;
    }
}
