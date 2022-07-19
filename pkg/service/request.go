package service

import (
	"encoding/binary"
	"github.com/vmihailenco/msgpack"
	"log"
	"nmid-v2/pkg/conf"
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
	instanceMapVal, err := Struct2Map(req.ScInfo.Instance)
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
	ingressUrlLen := uint32(len(req.ScInfo.IngressUrl))
	instanceLen := uint32(len(instancePackVal))
	req.DataLen = conf.UINT32_SIZE + serviceIdLen +
		conf.UINT32_SIZE + ingressUrlLen +
		conf.UINT32_SIZE + instanceLen
	contentLen = req.DataLen

	content = make([]byte, contentLen)
	binary.BigEndian.PutUint32(content[:conf.UINT32_SIZE], serviceIdLen)
	start := conf.UINT32_SIZE
	end := conf.UINT32_SIZE + conf.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], serviceIdLen)
	start = end
	end = start + conf.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], ingressUrlLen)
	start = end
	end = start + int(instanceLen)
	copy(content[start:end], instancePackVal)
	req.Data = content

	return
}

//EncodePack 打包
func (req *Request) EncodePack() (data []byte) {
	len := conf.MIN_DATA_SIZE + req.DataLen //add 12 bytes head
	data = GetBuffer(int(len))

	binary.BigEndian.PutUint32(data[:4], conf.CONN_TYPE_SERVICE)
	binary.BigEndian.PutUint32(data[4:8], req.DataType)
	binary.BigEndian.PutUint32(data[8:conf.MIN_DATA_SIZE], req.DataLen)
	copy(data[conf.MIN_DATA_SIZE:], req.Data)

	return
}
