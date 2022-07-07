package service

import (
	"encoding/binary"
	"fmt"
)

type Response struct {
	DataType uint32
	Data     []byte
	DataLen  uint32
}

func NewRes() (res *Response) {
	res = &Response{}
	return
}

func GetConnType(data []byte) (connType uint32) {
	if len(data) == 0 {
		return 0
	}

	if len(data) < 4 {
		return 0
	}

	connType = uint32(binary.BigEndian.Uint32(data[:4]))

	return
}

//DecodePack 解包
func DecodePack(data []byte) (resp *Response, resLen int, err error) {
	resLen = len(data)
	if resLen < MIN_DATA_SIZE {
		err = fmt.Errorf("InvalidData1: %v", data)
		return
	}
	cl := int(binary.BigEndian.Uint32(data[8:MIN_DATA_SIZE]))
	if resLen < MIN_DATA_SIZE+cl {
		err = fmt.Errorf("InvalidData2: %v", data)
		return
	}
	content := data[MIN_DATA_SIZE : MIN_DATA_SIZE+cl]
	if len(content) != cl {
		err = fmt.Errorf("InvalidData3: %v", data)
		return
	}

	resp = NewRes()
	resp.DataType = binary.BigEndian.Uint32(data[4:8])
	resp.DataLen = uint32(cl)
	resp.Data = content

	return
}
