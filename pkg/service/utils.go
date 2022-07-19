package service

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"time"
)

func GetBuffer(n int) (buf []byte) {
	buf = make([]byte, n)
	return
}

//GenServiceId generate service id
func GenServiceId(salt string) string {
	md5Ctx := md5.New()
	timeStr := strconv.FormatInt(int64(time.Now().Nanosecond()), 10)
	val := salt + timeStr

	io.WriteString(md5Ctx, val)
	md5Str := md5Ctx.Sum(nil)

	return hex.EncodeToString(md5Str)
}

func Struct2Map(content interface{}) (map[string]interface{}, error) {
	var name map[string]interface{}

	if marshalContent, err := json.Marshal(content); err != nil {
		return nil, err
	} else {
		d := json.NewDecoder(bytes.NewReader(marshalContent))
		d.UseNumber() // 设置将float64转为一个number
		if err := d.Decode(&name); err != nil {
			return nil, err
		} else {
			for k, v := range name {
				name[k] = v
			}
		}
	}

	return name, nil
}
