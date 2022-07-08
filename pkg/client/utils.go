package client

import "nmid-v2/pkg/conf"

func GetBuffer(n int) (buf []byte) {
	buf = make([]byte, n)
	return
}

func GetRetStruct() *conf.RetStruct {
	return &conf.RetStruct{
		Code: 0,
		Msg:  "",
		Data: make([]byte, 0),
	}
}
