<?php
namespace Nmid;

// Marker class: wraps a PHP string so Msgpack::encode() emits it as a msgpack
// bin type (0xc4/0xc5/0xc6) rather than a str type. Used for the binary
// "Data" field of RetStruct, mirroring node's Buffer.alloc() / Go's []byte.
class Bin
{
    public string $data;

    public function __construct(string $data)
    {
        $this->data = $data;
    }
}
