package service

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"strconv"
	"time"
)

func GetBuffer(n int) (buf []byte) {
	buf = make([]byte, n)
	return
}

//GenServiceId generate service id
func GenServiceId(ServiceName, ServiceHost string) string {
	md5Ctx := md5.New()
	timeStr := strconv.FormatInt(int64(time.Now().Nanosecond()), 10)
	val := ServiceName + ServiceHost + timeStr
	//md5Ctx.Write([]byte(val))

	io.WriteString(md5Ctx, val)
	md5Str := md5Ctx.Sum(nil)

	return hex.EncodeToString(md5Str)
}
