<?php
namespace Nmid;

// Standard nmid return struct: { Code, Msg, Data }.
// Matches node's buildRetStruct, Go's wor.GetRetStruct() and the java SDK.
// Data is encoded as a msgpack bin type (via Nmid\Bin) so the server/worker
// receive raw bytes, not a utf-8 string.
class RetStruct
{
    public static function build(int $code, string $msg, string $data = ''): string
    {
        $obj = [
            'Code' => $code,
            'Msg'  => $msg,
            'Data' => new Bin($data),
        ];
        return Msgpack::encode($obj);
    }

    // Decode a msgpack-encoded ret into [code, msg, data].
    public static function parse(string $ret): ?array
    {
        $v = Msgpack::decode($ret);
        if (!is_array($v)) {
            return null;
        }
        return [
            isset($v['Code']) ? (int)$v['Code'] : 0,
            isset($v['Msg']) ? (string)$v['Msg'] : '',
            isset($v['Data']) ? (string)$v['Data'] : '',
        ];
    }
}
