<?php
namespace Nmid\Client;

use Nmid\Constants;

// Error type for protocol-level failures (PDT_ERROR / PDT_CANT_DO /
// PDT_RATELIMIT). Mirrors node's PDT_ERROR handling and java's NmidError.
// Callers can catch NmidError and retry on PDT_CANT_DO ("have no job do").
class NmidError extends \RuntimeException
{
    public string $errorCode;

    private function __construct(string $code, string $message)
    {
        parent::__construct($message);
        $this->errorCode = $code;
    }

    public static function fromDataType(int $dataType): ?NmidError
    {
        return match ($dataType) {
            Constants::PDT_ERROR     => new self('PDT_ERROR', 'request error'),
            Constants::PDT_CANT_DO   => new self('PDT_CANT_DO', 'have no job do'),
            Constants::PDT_RATELIMIT => new self('PDT_RATELIMIT', 'have ratelimit'),
            default                  => null,
        };
    }
}
