package worker

import (
	"encoding/json"
	"github.com/vmihailenco/msgpack"
	"log"
	"nmid-v2/pkg/conf"
)

func GetBuffer(n int) (buf []byte) {
	buf = make([]byte, n)
	return
}

func MsgpackParamsMap(params []byte) map[string]interface{} {
	paramsMap := make(map[string]interface{})

	err := msgpack.Unmarshal(params, &paramsMap)
	if err != nil {
		log.Println("msgpack unmarshal error:", err)
		return nil
	}

	return paramsMap
}

func JsonParamsMap(params []byte) map[string]interface{} {
	paramsMap := make(map[string]interface{})

	err := json.Unmarshal(params, &paramsMap)
	if err != nil {
		log.Println("msgpack unmarshal error:", err)
		return nil
	}

	return paramsMap
}

func GetRetStruct() *conf.RetStruct {
	return &conf.RetStruct{
		Code: 0,
		Msg:  "",
		Data: make([]byte, 0),
	}
}
