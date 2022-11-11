package worker

import (
	"encoding/binary"
	"fmt"
	"nmid/pkg/model"
	"nmid/pkg/utils"
)

type Response struct {
	DataType uint32
	Data     []byte
	DataLen  uint32

	Handle           string
	HandleLen        uint32
	ParamsType       uint32
	ParamsHandleType uint32
	ParamsLen        uint32
	Params           []byte
	ParamsMap        map[string]interface{}
	JobId            string
	JobIdLen         uint32
	Ret              []byte
	RetLen           uint32

	Agent *Agent
}

func NewRes() (res *Response) {
	res = &Response{
		Data:       make([]byte, 0),
		ParamsType: model.PARAMS_TYPE_MSGPACK,
		Ret:        make([]byte, 0),
	}
	return
}

// DecodePack 解包
func DecodePack(data []byte) (resp *Response, resLen int, err error) {
	resLen = len(data)
	if resLen < model.MIN_DATA_SIZE {
		err = fmt.Errorf("invalid data: %v", data)
		return
	}
	cl := int(binary.BigEndian.Uint32(data[8:model.MIN_DATA_SIZE]))
	if resLen < model.MIN_DATA_SIZE+cl {
		err = fmt.Errorf("invalid data: %v", data)
		return
	}
	content := data[model.MIN_DATA_SIZE : model.MIN_DATA_SIZE+cl]
	if len(content) != cl {
		err = fmt.Errorf("invalid data: %v", data)
		return
	}

	resp = NewRes()
	resp.DataType = binary.BigEndian.Uint32(data[4:8])
	resp.DataLen = uint32(cl)
	resp.Data = content

	if resp.DataType == model.PDT_S_GET_DATA {
		//新的解包协议
		start := model.MIN_DATA_SIZE
		end := model.MIN_DATA_SIZE + model.UINT32_SIZE
		resp.ParamsType = uint32(binary.BigEndian.Uint32(data[start:end]))
		start = end
		end = start + model.UINT32_SIZE
		resp.ParamsHandleType = uint32(binary.BigEndian.Uint32(data[start:end]))
		start = end
		end = start + model.UINT32_SIZE
		resp.HandleLen = binary.BigEndian.Uint32(data[start:end])
		start = end
		end = start + model.UINT32_SIZE
		resp.ParamsLen = binary.BigEndian.Uint32(data[start:end])
		start = end
		end = start + model.UINT32_SIZE
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
	if resp.ParamsType == model.PARAMS_TYPE_MSGPACK {
		resp.ParamsMap = utils.MsgpackParamsMap(params)
	} else if resp.ParamsType == model.PARAMS_TYPE_JSON {
		resp.ParamsMap = utils.JsonParamsMap(params)
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
