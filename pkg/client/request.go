package client

import (
	"encoding/binary"
	"nmid-v2/pkg/conf"
)

type Request struct {
	DataType uint32
	Data     []byte
	DataLen  uint32

	Handle           string
	HandleLen        uint32
	ParamsType       uint32
	ParamsHandleType uint32
	ParamsLen        uint32
	Params           []byte
	Ret              []byte
	RetLen           uint32
}

func NewReq() (req *Request) {
	req = &Request{
		Data:             make([]byte, 0),
		ParamsType:       conf.PARAMS_TYPE_MSGPACK,
		ParamsHandleType: conf.PARAMS_HANDLE_TYPE_ENCODE,
		Ret:              make([]byte, 0),
	}
	return
}

//ContentPack 打包内容
func (req *Request) ContentPack(dataType uint32, handle string, params []byte) (content []byte, contentLen uint32) {
	req.DataType = dataType
	req.Handle = handle
	req.HandleLen = uint32(len(handle))
	req.Params = params
	req.ParamsLen = uint32(len(params))
	req.DataLen = uint32(conf.UINT32_SIZE + conf.UINT32_SIZE + conf.UINT32_SIZE + req.HandleLen + conf.UINT32_SIZE + req.ParamsLen)
	contentLen = req.DataLen

	content = make([]byte, contentLen)
	binary.BigEndian.PutUint32(content[:conf.UINT32_SIZE], req.ParamsType)
	start := conf.UINT32_SIZE
	end := start + conf.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], req.ParamsHandleType)
	start = end
	end = start + conf.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], req.HandleLen)
	start = end
	end = start + int(req.HandleLen)
	copy(content[start:end], []byte(req.Handle))
	start = end
	end = start + conf.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], req.ParamsLen)
	start = end
	end = start + int(req.ParamsLen)
	copy(content[start:end], req.Params)
	req.Data = content

	return
}

//EncodePack 打包
func (req *Request) EncodePack() (data []byte) {
	len := conf.MIN_DATA_SIZE + req.DataLen //add 12 bytes head
	data = GetBuffer(int(len))

	binary.BigEndian.PutUint32(data[:4], conf.CONN_TYPE_CLIENT)
	binary.BigEndian.PutUint32(data[4:8], req.DataType)
	binary.BigEndian.PutUint32(data[8:conf.MIN_DATA_SIZE], req.DataLen)
	copy(data[conf.MIN_DATA_SIZE:], req.Data)

	return
}
