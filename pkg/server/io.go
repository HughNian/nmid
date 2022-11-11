package server

import (
	"encoding/binary"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/service"
	"github.com/HughNian/nmid/pkg/utils"
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
	JobId            string
	JobIdLen         uint32
	Ret              []byte
	RetLen           uint32

	ScInfo service.ServiceInfo
}

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
	JobId            string
	JobIdLen         uint32
	Ret              []byte
	RetLen           uint32
}

func NewReq() (req *Request) {
	req = &Request{
		Data: make([]byte, 0),
		Ret:  make([]byte, 0),
	}
	return
}

func NewRes() (res *Response) {
	res = &Response{
		Data: make([]byte, 0),
		Ret:  make([]byte, 0),
	}
	return
}

func (req *Request) GetReqDataType() uint32 {
	return req.DataType
}

func (res *Response) GetResDataType() uint32 {
	return res.DataType
}

func (req *Request) GetReqData() []byte {
	if req.DataLen == uint32(len(req.Data)) {
		return req.Data
	}

	return []byte(``)
}

func (res *Response) GetResData() []byte {
	if res.DataLen == uint32(len(res.Data)) {
		return res.Data
	}

	return []byte(``)
}

func (res *Response) GetResHandle() string {
	return res.Handle
}

//GetResContent 打包内容
func (res *Response) GetResContent() (content []byte, contentLen int) {
	if res.DataType == model.PDT_S_GET_DATA {
		contentLen = int(model.UINT32_SIZE + model.UINT32_SIZE + model.UINT32_SIZE + res.HandleLen + model.UINT32_SIZE + res.ParamsLen + model.UINT32_SIZE + res.JobIdLen)
		content = utils.GetBuffer(contentLen)

		//新的发给worker的打包协议
		binary.BigEndian.PutUint32(content[:model.UINT32_SIZE], res.ParamsType)
		start := model.UINT32_SIZE
		end := start + model.UINT32_SIZE
		binary.BigEndian.PutUint32(content[start:end], res.ParamsHandleType)
		start = end
		end = start + model.UINT32_SIZE
		binary.BigEndian.PutUint32(content[start:end], uint32(res.HandleLen))
		start = end
		end = start + model.UINT32_SIZE
		binary.BigEndian.PutUint32(content[start:end], uint32(res.ParamsLen))
		start = end
		end = start + model.UINT32_SIZE
		binary.BigEndian.PutUint32(content[start:end], uint32(res.JobIdLen))
		start = end
		end = start + int(res.HandleLen)
		copy(content[start:end], []byte(res.Handle))
		start = end
		end = start + int(res.ParamsLen)
		copy(content[start:end], res.Params)
		start = end
		copy(content[start:], res.JobId)
	} else if res.DataType == model.PDT_S_RETURN_DATA {
		contentLen = int(model.UINT32_SIZE + res.HandleLen + model.UINT32_SIZE + res.ParamsLen + model.UINT32_SIZE + res.RetLen)
		content = utils.GetBuffer(contentLen)

		//新的发给client的打包协议
		binary.BigEndian.PutUint32(content[:model.UINT32_SIZE], uint32(res.HandleLen))
		start := model.UINT32_SIZE
		end := start + model.UINT32_SIZE
		binary.BigEndian.PutUint32(content[start:end], uint32(res.ParamsLen))
		start = end
		end = start + model.UINT32_SIZE
		binary.BigEndian.PutUint32(content[start:end], uint32(res.RetLen))
		start = end
		end = start + int(res.HandleLen)
		copy(content[start:end], res.Handle)
		start = end
		end = start + int(res.ParamsLen)
		copy(content[start:end], res.Params)
		start = end
		copy(content[start:], res.Ret)
	} else if res.DataType == model.PDT_NO_JOB || res.DataType == model.PDT_OK || res.DataType == model.PDT_ERROR || res.DataType == model.PDT_CANT_DO {
		content = []byte(``)
		contentLen = 0
	}

	return
}

//ReqDecodePack 解包
func (req *Request) ReqDecodePack() {
	if req.DataLen > 0 && len(req.Data) > 0 && req.DataLen == uint32(len(req.Data)) {
		if req.DataType == model.PDT_W_RETURN_DATA {
			var handle []byte
			var handLen int
			req.HandleLen = uint32(binary.BigEndian.Uint32(req.Data[:model.UINT32_SIZE]))
			handLen = int(req.HandleLen)
			handle = utils.GetBuffer(handLen)
			start := model.UINT32_SIZE
			end := model.UINT32_SIZE + handLen
			copy(handle, req.Data[start:end])
			req.Handle = string(handle)

			var params []byte
			var paramsLen int
			start = end
			end = start + model.UINT32_SIZE
			req.ParamsLen = uint32(binary.BigEndian.Uint32(req.Data[start:end]))
			paramsLen = int(req.ParamsLen)
			params = utils.GetBuffer(paramsLen)
			start = end
			end = start + paramsLen
			copy(params, req.Data[start:end])
			req.Params = params //append(req.Params, params...)

			var ret []byte
			var retLen int
			start = end
			end = start + model.UINT32_SIZE
			req.RetLen = uint32(binary.BigEndian.Uint32(req.Data[start:end]))
			retLen = int(req.RetLen)
			ret = utils.GetBuffer(retLen)
			start = end
			end = start + retLen
			copy(ret, req.Data[start:end])
			req.Ret = ret //append(req.Ret, ret...)

			var jobId []byte
			var jobIdLen int
			start = end
			end = start + model.UINT32_SIZE
			req.JobIdLen = uint32(binary.BigEndian.Uint32(req.Data[start:end]))
			jobIdLen = int(req.JobIdLen)
			jobId = utils.GetBuffer(jobIdLen)
			start = end
			copy(jobId, req.Data[start:])
			req.JobId = string(jobId)
		} else if req.DataType == model.PDT_C_DO_JOB {
			var handle []byte
			var handLen int

			req.ParamsType = uint32(binary.BigEndian.Uint32(req.Data[:model.UINT32_SIZE]))
			start := model.UINT32_SIZE
			end := start + model.UINT32_SIZE
			req.ParamsHandleType = uint32(binary.BigEndian.Uint32(req.Data[start:end]))
			start = end
			end = start + model.UINT32_SIZE
			req.HandleLen = uint32(binary.BigEndian.Uint32(req.Data[start:end]))
			handLen = int(req.HandleLen)
			handle = utils.GetBuffer(handLen)
			start = end
			end = start + handLen
			copy(handle, req.Data[start:end])
			req.Handle = string(handle)

			var params []byte
			var paramsLen int
			start = end
			end = start + model.UINT32_SIZE
			req.ParamsLen = uint32(binary.BigEndian.Uint32(req.Data[start:end]))
			paramsLen = int(req.ParamsLen)
			params = utils.GetBuffer(paramsLen)
			start = end
			copy(params, req.Data[start:])
			req.Params = params //append(req.Params, params...)
		} else if req.DataType == model.PDT_SC_REG_SERVICE {

		}
	}
}

//ResEncodePack 打包
func (res *Response) ResEncodePack() (resData []byte) {
	content, contentLen := res.GetResContent()
	// fmt.Println("######content-", content)
	// fmt.Println("######contentLen-", contentLen)

	resDataLen := model.MIN_DATA_SIZE + contentLen //数据内容长度
	res.DataLen = uint32(resDataLen)
	// fmt.Println("######resDataLen-", resDataLen)

	resData = utils.GetBuffer(resDataLen)
	binary.BigEndian.PutUint32(resData[:model.UINT32_SIZE], model.CONN_TYPE_SERVER)
	binary.BigEndian.PutUint32(resData[model.UINT32_SIZE:8], res.DataType)
	binary.BigEndian.PutUint32(resData[8:model.MIN_DATA_SIZE], uint32(contentLen))

	if contentLen > 0 {
		copy(resData[model.MIN_DATA_SIZE:], content)
		res.Data = resData //append(res.Data, resData...)
	}

	return
}
