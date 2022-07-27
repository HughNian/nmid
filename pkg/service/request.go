package service

import (
	"encoding/binary"
	"github.com/vmihailenco/msgpack"
	"log"
	"nmid-v2/pkg/model"
	"nmid-v2/pkg/utils"
)

type Request struct {
	DataType uint32
	Data     []byte
	DataLen  uint32

	ScInfo *ServiceInfo
}

func NewReq(scInfo *ServiceInfo) (req *Request) {
	req = &Request{
		ScInfo: scInfo,
	}
	return
}

//ServiceInfoPack service信息打包内容
func (req *Request) ServiceInfoPack(dataType uint32) (content []byte, contentLen uint32) {
	instanceMapVal, err := utils.Struct2Map(req.ScInfo.Instance)
	if nil != err {
		log.Fatalln("ServiceInfoPack instance map val err", err)
		return
	}
	instancePackVal, err := msgpack.Marshal(instanceMapVal)
	if nil != err {
		log.Println("ServiceInfoPack instance pack val err", err)
		return
	}

	req.DataType = dataType
	serviceIdLen := uint32(len(req.ScInfo.ServiceId))
	ingressUrlLen := uint32(len(req.ScInfo.InFlowUrl))
	instanceLen := uint32(len(instancePackVal))
	req.DataLen = model.UINT32_SIZE + serviceIdLen +
		model.UINT32_SIZE + ingressUrlLen +
		model.UINT32_SIZE + instanceLen
	contentLen = req.DataLen

	content = make([]byte, contentLen)
	binary.BigEndian.PutUint32(content[:model.UINT32_SIZE], serviceIdLen)
	start := model.UINT32_SIZE
	end := model.UINT32_SIZE + model.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], serviceIdLen)
	start = end
	end = start + model.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], ingressUrlLen)
	start = end
	end = start + int(instanceLen)
	copy(content[start:end], instancePackVal)
	req.Data = content

	return
}

//EncodePack 打包
func (req *Request) EncodePack() (data []byte) {
	len := model.MIN_DATA_SIZE + req.DataLen //add 12 bytes head
	data = utils.GetBuffer(int(len))

	binary.BigEndian.PutUint32(data[:4], model.CONN_TYPE_SERVICE)
	binary.BigEndian.PutUint32(data[4:8], req.DataType)
	binary.BigEndian.PutUint32(data[8:model.MIN_DATA_SIZE], req.DataLen)
	copy(data[model.MIN_DATA_SIZE:], req.Data)

	return
}
