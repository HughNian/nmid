package worker

import (
	"encoding/binary"
	"fmt"
	"nmid-v2/pkg/conf"
)

type Response struct {
	DataType uint32
	Data     []byte
	DataLen  uint32

	Handle     string
	HandleLen  uint32
	ParamsType uint32
	ParamsNum  uint32
	ParamsLen  uint32
	Params     []byte
	ParamsMap  map[string]interface{}
	JobId      string
	JobIdLen   uint32
	Ret        []byte
	RetLen     uint32

	Agent *Agent
}

func NewRes() (res *Response) {
	res = &Response{
		Data:       make([]byte, 0),
		DataLen:    0,
		Handle:     ``,
		HandleLen:  0,
		ParamsType: conf.PARAMS_TYPE_MSGPACK,
		ParamsNum:  0,
		ParamsLen:  0,
		Params:     make([]byte, 0),
		Ret:        make([]byte, 0),
		RetLen:     0,
	}
	return
}

//DecodePack 解包
func DecodePack(data []byte) (resp *Response, resLen int, err error) {
	resLen = len(data)
	if resLen < conf.MIN_DATA_SIZE {
		err = fmt.Errorf("invalid data: %v", data)
		return
	}
	cl := int(binary.BigEndian.Uint32(data[8:conf.MIN_DATA_SIZE]))
	if resLen < conf.MIN_DATA_SIZE+cl {
		err = fmt.Errorf("invalid data: %v", data)
		return
	}
	content := data[conf.MIN_DATA_SIZE : conf.MIN_DATA_SIZE+cl]
	if len(content) != cl {
		err = fmt.Errorf("invalid data: %v", data)
		return
	}

	resp = NewRes()
	resp.DataType = binary.BigEndian.Uint32(data[4:8])
	resp.DataLen = uint32(cl)
	resp.Data = content

	if resp.DataType == conf.PDT_S_GET_DATA {
		//新的解包协议
		start := conf.MIN_DATA_SIZE
		end := conf.MIN_DATA_SIZE + conf.UINT32_SIZE
		resp.HandleLen = binary.BigEndian.Uint32(data[start:end])
		start = end
		end = start + conf.UINT32_SIZE
		resp.ParamsLen = binary.BigEndian.Uint32(data[start:end])
		start = end
		end = start + conf.UINT32_SIZE
		resp.JobIdLen = binary.BigEndian.Uint32(data[start:end])
		start = end
		end = start + int(resp.HandleLen)
		resp.Handle = string(data[start:end])
		start = end
		end = start + int(resp.ParamsLen)
		resp.ParseParams(data[start:end])
		start = end
		end = start + int(resp.JobIdLen)
		resp.JobId = string(data[start:end])
	}

	return
}

func (resp *Response) GetResponse() *Response {
	return resp
}

func (resp *Response) ParseParams(params []byte) {
	resp.Params = params
	if resp.ParamsType == conf.PARAMS_TYPE_MSGPACK {
		resp.ParamsMap = MsgpackParamsMap(params)
	} else if resp.ParamsType == conf.PARAMS_TYPE_JSON {
		resp.ParamsMap = JsonParamsMap(params)
	}
}

func (resp *Response) GetParams() []byte {
	if resp.ParamsLen == 0 {
		return nil
	}

	return resp.Params
}

func (resp *Response) GetParamsMap() map[string]interface{} {
	if resp.ParamsLen == 0 {
		return nil
	}

	return resp.ParamsMap
}
