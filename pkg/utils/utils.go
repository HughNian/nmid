package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/vmihailenco/msgpack"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func GetBuffer(n int) (buf []byte) {
	buf = make([]byte, n)
	return
}

func GetId() string {
	value := int64(time.Now().Nanosecond()) << 32
	next := atomic.AddInt64(&value, 1)
	return strconv.FormatInt(next, 10)
}

func GetJobId(Handle, Params string) string {
	md5Ctx := md5.New()
	timeStr := strconv.FormatInt(int64(time.Now().Nanosecond()), 10)
	val := Handle + Params + timeStr
	//md5Ctx.Write([]byte(val))

	io.WriteString(md5Ctx, val)
	md5Str := md5Ctx.Sum(nil)

	return hex.EncodeToString(md5Str)
}

func IsMulParams(params []byte) bool {
	var decParams []string
	err := msgpack.Unmarshal(params, &decParams)
	if err == nil {
		plen := len(decParams)
		if plen > 1 {
			return true
		} else {
			return false
		}
	}

	return false
}

// GenServiceId generate service id
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
		log.Println("json unmarshal error:", err)
		return nil
	}

	return paramsMap
}

func OsPath(path string) string {
	if runtime.GOOS == "windows" {
		return "file:////%3F/" + filepath.ToSlash(path)
	}

	return path
}

type BufferPool struct {
	pool sync.Pool
}

func NewBufferPool() *BufferPool {
	bp := BufferPool{}
	bp.pool.New = func() interface{} {
		b := make([]byte, 32*1024)
		return b
	}
	return &bp
}

func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

func (bp *BufferPool) Put(v []byte) {
	bp.pool.Put(v)
}

func PathExist(path string) bool {
	_, err := os.Stat(path)

	if nil != err {
		if os.IsExist(err) {
			return true
		}

		return false
	}

	return true
}

func CreateFile(name string) (*os.File, error) {
	err := os.MkdirAll(string([]rune(name)[0:strings.LastIndex(name, "/")]), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(name)
}
