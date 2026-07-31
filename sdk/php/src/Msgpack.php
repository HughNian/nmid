<?php
namespace Nmid;

// Minimal MessagePack encoder/decoder.
// Only the types the nmid SDK needs are supported: null, bool, int, str, bin,
// array (list) and map (associative). This keeps the SDK dependency-free.
//
// A Nmid\Bin value is encoded as a bin type; plain strings are encoded as str.
class Msgpack
{
    public static function encode($value): string
    {
        $out = '';
        self::encodeValue($value, $out);
        return $out;
    }

    public static function decode(string $data)
    {
        $offset = 0;
        return self::decodeValue($data, $offset);
    }

    // ---- encode ----

    public static function encodeValue($value, string &$out): void
    {
        if ($value === null) {
            $out .= "\xc0";
        } elseif (is_bool($value)) {
            $out .= $value ? "\xc3" : "\xc2";
        } elseif ($value instanceof Bin) {
            self::encodeBin($value->data, $out);
        } elseif (is_int($value)) {
            self::encodeInt($value, $out);
        } elseif (is_string($value)) {
            self::encodeStr($value, $out);
        } elseif (is_array($value)) {
            if (self::isList($value)) {
                self::encodeArray($value, $out);
            } else {
                self::encodeMap($value, $out);
            }
        } else {
            // Fallback: stringify
            self::encodeStr((string)$value, $out);
        }
    }

    private static function encodeInt(int $value, string &$out): void
    {
        if ($value >= 0 && $value <= 0x7f) {
            $out .= chr($value);
        } elseif ($value < 0 && $value >= -32) {
            $out .= chr($value & 0xff);
        } elseif ($value >= 0 && $value <= 0xff) {
            $out .= "\xcc" . chr($value);
        } elseif ($value >= 0 && $value <= 0xffff) {
            $out .= "\xcd" . pack('n', $value);
        } elseif ($value >= 0 && $value <= 0xffffffff) {
            $out .= "\xce" . pack('N', $value);
        } elseif ($value < 0 && $value >= -128) {
            $out .= "\xd0" . pack('c', $value);
        } elseif ($value < 0 && $value >= -32768) {
            $out .= "\xd1" . pack('n', $value & 0xffff);
        } elseif ($value < 0 && $value >= -2147483648) {
            $out .= "\xd2" . pack('N', $value & 0xffffffff);
        } else {
            // int64
            $out .= "\xd3" . pack('J', $value);
        }
    }

    private static function encodeStr(string $value, string &$out): void
    {
        $len = strlen($value);
        if ($len <= 31) {
            $out .= chr(0xa0 | $len) . $value;
        } elseif ($len <= 0xff) {
            $out .= "\xd9" . chr($len) . $value;
        } elseif ($len <= 0xffff) {
            $out .= "\xda" . pack('n', $len) . $value;
        } else {
            $out .= "\xdb" . pack('N', $len) . $value;
        }
    }

    private static function encodeBin(string $value, string &$out): void
    {
        $len = strlen($value);
        if ($len <= 0xff) {
            $out .= "\xc4" . chr($len) . $value;
        } elseif ($len <= 0xffff) {
            $out .= "\xc5" . pack('n', $len) . $value;
        } else {
            $out .= "\xc6" . pack('N', $len) . $value;
        }
    }

    private static function encodeArray(array $value, string &$out): void
    {
        $n = count($value);
        if ($n <= 15) {
            $out .= chr(0x90 | $n);
        } elseif ($n <= 0xffff) {
            $out .= "\xdc" . pack('n', $n);
        } else {
            $out .= "\xdd" . pack('N', $n);
        }
        foreach ($value as $v) {
            self::encodeValue($v, $out);
        }
    }

    private static function encodeMap(array $value, string &$out): void
    {
        $n = count($value);
        if ($n <= 15) {
            $out .= chr(0x80 | $n);
        } elseif ($n <= 0xffff) {
            $out .= "\xde" . pack('n', $n);
        } else {
            $out .= "\xdf" . pack('N', $n);
        }
        foreach ($value as $k => $v) {
            self::encodeValue((string)$k, $out);
            self::encodeValue($v, $out);
        }
    }

    private static function isList(array $arr): bool
    {
        if (empty($arr)) {
            return true;
        }
        $i = 0;
        foreach ($arr as $k => $_) {
            if ($k !== $i) {
                return false;
            }
            $i++;
        }
        return true;
    }

    // ---- decode ----

    private static function decodeValue(string $data, int &$offset)
    {
        if ($offset >= strlen($data)) {
            return null;
        }
        $b = ord($data[$offset]);
        $offset++;

        if ($b <= 0x7f) {
            return $b; // positive fixint
        }
        if ($b >= 0xe0) {
            return $b - 0x100; // negative fixint
        }
        if ($b >= 0x80 && $b <= 0x8f) {
            return self::decodeMap($data, $offset, $b & 0x0f); // fixmap
        }
        if ($b >= 0x90 && $b <= 0x9f) {
            return self::decodeArray($data, $offset, $b & 0x0f); // fixarray
        }
        if ($b >= 0xa0 && $b <= 0xbf) {
            return self::readStr($data, $offset, $b & 0x1f); // fixstr
        }

        switch ($b) {
            case 0xc0: return null;
            case 0xc2: return false;
            case 0xc3: return true;
            case 0xc4: // bin8
                $len = ord($data[$offset++]);
                return self::readStr($data, $offset, $len);
            case 0xc5: // bin16
                $len = unpack('n', substr($data, $offset, 2))[1];
                $offset += 2;
                return self::readStr($data, $offset, $len);
            case 0xc6: // bin32
                $len = unpack('N', substr($data, $offset, 4))[1];
                $offset += 4;
                return self::readStr($data, $offset, $len);
            case 0xcc: // uint8
                return ord($data[$offset++]);
            case 0xcd: // uint16
                $v = unpack('n', substr($data, $offset, 2))[1];
                $offset += 2;
                return $v;
            case 0xce: // uint32
                $v = unpack('N', substr($data, $offset, 4))[1];
                $offset += 4;
                return $v;
            case 0xcf: // uint64
                $v = unpack('J', substr($data, $offset, 8))[1];
                $offset += 8;
                return $v;
            case 0xd0: // int8
                $v = unpack('c', substr($data, $offset, 1))[1];
                $offset++;
                return $v;
            case 0xd1: // int16
                $v = unpack('n', substr($data, $offset, 2))[1];
                if ($v >= 0x8000) {
                    $v -= 0x10000;
                }
                $offset += 2;
                return $v;
            case 0xd2: // int32
                $v = unpack('N', substr($data, $offset, 4))[1];
                if ($v >= 0x80000000) {
                    $v -= 0x100000000;
                }
                $offset += 4;
                return $v;
            case 0xd3: // int64
                $v = unpack('J', substr($data, $offset, 8))[1];
                $offset += 8;
                return $v;
            case 0xd9: // str8
                $len = ord($data[$offset++]);
                return self::readStr($data, $offset, $len);
            case 0xda: // str16
                $len = unpack('n', substr($data, $offset, 2))[1];
                $offset += 2;
                return self::readStr($data, $offset, $len);
            case 0xdb: // str32
                $len = unpack('N', substr($data, $offset, 4))[1];
                $offset += 4;
                return self::readStr($data, $offset, $len);
            case 0xdc: // array16
                $n = unpack('n', substr($data, $offset, 2))[1];
                $offset += 2;
                return self::decodeArray($data, $offset, $n);
            case 0xdd: // array32
                $n = unpack('N', substr($data, $offset, 4))[1];
                $offset += 4;
                return self::decodeArray($data, $offset, $n);
            case 0xde: // map16
                $n = unpack('n', substr($data, $offset, 2))[1];
                $offset += 2;
                return self::decodeMap($data, $offset, $n);
            case 0xdf: // map32
                $n = unpack('N', substr($data, $offset, 4))[1];
                $offset += 4;
                return self::decodeMap($data, $offset, $n);
            default:
                return null;
        }
    }

    private static function decodeMap(string $data, int &$offset, int $n): array
    {
        $map = [];
        for ($i = 0; $i < $n; $i++) {
            $k = self::decodeValue($data, $offset);
            $v = self::decodeValue($data, $offset);
            $map[$k] = $v;
        }
        return $map;
    }

    private static function decodeArray(string $data, int &$offset, int $n): array
    {
        $arr = [];
        for ($i = 0; $i < $n; $i++) {
            $arr[] = self::decodeValue($data, $offset);
        }
        return $arr;
    }

    private static function readStr(string $data, int &$offset, int $len): string
    {
        if ($len <= 0) {
            return '';
        }
        $s = substr($data, $offset, $len);
        $offset += $len;
        return $s;
    }
}
