package worker

import (
	"encoding/binary"

	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/utils"
)

type Request struct {
	DataType uint32
	Data     []byte
	DataLen  uint32

	Handle     string
	HandleLen  uint32
	ParamsType uint32
	ParamsLen  uint32
	Params     []byte
	JobId      string
	JobIdLen   uint32
	Ret        []byte
	RetLen     uint32
}

func NewReq() (req *Request) {
	req = &Request{
		Data:      make([]byte, 0),
		DataLen:   0,
		Handle:    ``,
		HandleLen: 0,
		ParamsLen: 0,
		Params:    make([]byte, 0),
		Ret:       make([]byte, 0),
		RetLen:    0,
	}
	return
}

// HeartBeatPack 打包内容-心跳
func (req *Request) HeartBeatPack() (content []byte, err error) {
	data := "PING"

	req.DataType = model.PDT_W_HEARTBEAT_PING
	req.DataLen = uint32(len(data))
	req.Data = []byte(data)
	content = req.Data

	return
}

// SetWorkerName 打包内容-设置worker名称
func (req *Request) SetWorkerName(workerName string) (content []byte, err error) {
	req.DataType = model.PDT_W_SET_NAME
	req.DataLen = uint32(len(workerName))
	req.Data = []byte(workerName)
	content = req.Data

	return
}

// AddFunctionPack 打包内容-添加方法
func (req *Request) AddFunctionPack(funcName string) (content []byte, err error) {
	req.DataType = model.PDT_W_ADD_FUNC
	req.DataLen = uint32(len(funcName))
	req.Data = []byte(funcName)
	content = req.Data

	return
}

// DelFunctionPack 打包内容-删除方法
func (req *Request) DelFunctionPack(funcName string) (content []byte, err error) {
	req.DataType = model.PDT_W_DEL_FUNC
	req.DataLen = uint32(len(funcName))
	req.Data = []byte(funcName)
	content = req.Data

	return
}

// GrabDataPack 打包内容-抓取任务
func (req *Request) GrabDataPack() (content []byte, err error) {
	req.DataType = model.PDT_W_GRAB_JOB
	req.DataLen = 0
	req.Data = []byte(``)
	content = req.Data

	return
}

// WakeupPack 打包内容-唤醒
func (req *Request) WakeupPack() {
	req.DataType = model.PDT_WAKEUP
	req.DataLen = 0
	req.Data = []byte(``)
}

// LimitExceedPack 打包内容-限流
func (req *Request) LimitExceedPack() {
	req.DataType = model.PDT_RATELIMIT
	req.DataLen = 0
	req.Data = []byte(``)
}

// RetPack 打包内容-返回结果
func (req *Request) RetPack(ret []byte) (content []byte, err error) {
	req.Ret = ret
	req.RetLen = uint32(len(ret))

	req.DataType = model.PDT_W_RETURN_DATA
	req.DataLen = model.UINT32_SIZE + req.HandleLen + model.UINT32_SIZE + req.ParamsLen + model.UINT32_SIZE + req.RetLen + model.UINT32_SIZE + req.JobIdLen

	length := int(req.DataLen)
	content = utils.GetBuffer(length)
	binary.BigEndian.PutUint32(content[:model.UINT32_SIZE], req.HandleLen)
	start := model.UINT32_SIZE
	end := int(model.UINT32_SIZE + req.HandleLen)
	copy(content[start:end], []byte(req.Handle))
	start = end
	end = start + model.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], uint32(req.ParamsLen))
	start = end
	end = start + int(req.ParamsLen)
	copy(content[start:end], req.Params)
	start = end
	end = start + model.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], req.RetLen)
	start = end
	end = start + int(req.RetLen)
	copy(content[start:end], req.Ret)
	start = end
	end = start + model.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], req.JobIdLen)
	start = end
	end = start + int(req.JobIdLen)
	copy(content[start:end], req.JobId)
	req.Data = content

	return
}

// EncodePack 打包
func (req *Request) EncodePack() (data []byte) {
	len := model.MIN_DATA_SIZE + req.DataLen //add 12 bytes head
	data = utils.GetBuffer(int(len))

	binary.BigEndian.PutUint32(data[:4], model.CONN_TYPE_WORKER)
	binary.BigEndian.PutUint32(data[4:8], req.DataType)
	binary.BigEndian.PutUint32(data[8:model.MIN_DATA_SIZE], req.DataLen)
	copy(data[model.MIN_DATA_SIZE:], req.Data)

	return
}
