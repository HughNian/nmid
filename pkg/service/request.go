package service

import (
	"encoding/binary"
	"nmid-v2/pkg/conf"
)

type Request struct {
	DataType uint32
	Data     []byte
	DataLen  uint32

	ScInfo ScInfo
}

type ScInfo struct {
	ServiceId   string
	ServiceName string
	ServiceHost string
	ServicePort uint32
}

func NewReq(scInfo ScInfo) (req *Request) {
	req = &Request{
		ScInfo: scInfo,
	}
	return
}

//ServiceInfoPack service信息打包内容
func (req *Request) ServiceInfoPack(dataType uint32) (content []byte, contentLen uint32) {
	req.DataType = dataType
	serviceIdLen := uint32(len(req.ScInfo.ServiceId))
	serviceNameLen := uint32(len(req.ScInfo.ServiceName))
	serviceHostLen := uint32(len(req.ScInfo.ServiceHost))
	req.DataLen = conf.UINT32_SIZE + serviceIdLen + conf.UINT32_SIZE + serviceNameLen + conf.UINT32_SIZE + serviceHostLen + conf.UINT32_SIZE
	contentLen = req.DataLen

	content = make([]byte, contentLen)
	binary.BigEndian.PutUint32(content[:conf.UINT32_SIZE], serviceIdLen)
	start := conf.UINT32_SIZE
	end := conf.UINT32_SIZE + conf.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], serviceNameLen)
	start = end
	end = start + conf.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], serviceHostLen)
	start = end
	end = start + conf.UINT32_SIZE
	binary.BigEndian.PutUint32(content[start:end], req.ScInfo.ServicePort)
	start = end
	end = start + int(serviceIdLen)
	copy(content[start:end], req.ScInfo.ServiceId)
	start = end
	end = start + int(serviceNameLen)
	copy(content[start:end], req.ScInfo.ServiceName)
	start = end
	end = start + int(serviceHostLen)
	copy(content[start:end], req.ScInfo.ServiceHost)
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
