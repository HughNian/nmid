package direct

import (
	"encoding/binary"
	"errors"
)

const (
	FrameHeaderSize = 12
)

const (
	ConnTypeDirectClient = 101
	ConnTypeDirectServer = 102
)

const (
	DTCall   = 1
	DTReply  = 2
	DTCast   = 3
	DTHealth = 4
)

type Call struct {
	Service string
	Payload []byte
}

type Reply struct {
	Status uint32
	Data   []byte
}

func PackFrame(connType, dataType uint32, content []byte) []byte {
	out := make([]byte, FrameHeaderSize+len(content))
	binary.BigEndian.PutUint32(out[:4], connType)
	binary.BigEndian.PutUint32(out[4:8], dataType)
	binary.BigEndian.PutUint32(out[8:FrameHeaderSize], uint32(len(content)))
	copy(out[FrameHeaderSize:], content)
	return out
}

func PackCall(service string, payload []byte) []byte {
	s := []byte(service)
	out := make([]byte, 4+len(s)+4+len(payload))
	binary.BigEndian.PutUint32(out[:4], uint32(len(s)))
	copy(out[4:4+len(s)], s)
	off := 4 + len(s)
	binary.BigEndian.PutUint32(out[off:off+4], uint32(len(payload)))
	copy(out[off+4:], payload)
	return out
}

func UnpackCall(content []byte) (Call, error) {
	if len(content) < 8 {
		return Call{}, errors.New("invalid call")
	}
	slen := int(binary.BigEndian.Uint32(content[:4]))
	if slen < 0 || 4+slen+4 > len(content) {
		return Call{}, errors.New("invalid service length")
	}
	service := string(content[4 : 4+slen])
	off := 4 + slen
	plen := int(binary.BigEndian.Uint32(content[off : off+4]))
	off += 4
	if plen < 0 || off+plen > len(content) {
		return Call{}, errors.New("invalid payload length")
	}
	return Call{Service: service, Payload: content[off : off+plen]}, nil
}

func PackReply(status uint32, data []byte) []byte {
	out := make([]byte, 4+4+len(data))
	binary.BigEndian.PutUint32(out[:4], status)
	binary.BigEndian.PutUint32(out[4:8], uint32(len(data)))
	copy(out[8:], data)
	return out
}

func UnpackReply(content []byte) (Reply, error) {
	if len(content) < 8 {
		return Reply{}, errors.New("invalid reply")
	}
	status := binary.BigEndian.Uint32(content[:4])
	dlen := int(binary.BigEndian.Uint32(content[4:8]))
	if dlen < 0 || 8+dlen > len(content) {
		return Reply{}, errors.New("invalid reply length")
	}
	return Reply{Status: status, Data: content[8 : 8+dlen]}, nil
}

