package server

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/vmihailenco/msgpack"
	"gopkg.in/yaml.v2"
)

type ServerConfig struct {
	NETWORK string  `yaml:"NETWORK"`
	HOST    string  `yaml:"HOST"`
	PORT    string  `yaml:"PORT"`
	BREAKER Breaker `yaml:"BREAKER"`
}

type Breaker struct {
	SerialErrorNumbers uint32 `yaml:"SERIAL_ERROR_NUMBERS"`
	ErrorPercent       uint8  `yaml:"ERROR_PERCENT"`
	MaxRequest         uint32 `yaml:"MAX_REQUEST"`
	Interval           uint32 `yaml:"INTERVAL"`
	Timeout            uint32 `yaml:"TIMEOUT"`
	RuleType           uint8  `yaml:"RULE_TYPE"`
	Btype              int8   `yaml:"BTYPE"` // 熔断粒度path单个实例下接口级别，host单个实例级别，默认接口级别
	RequestTimeout     uint32 `yaml:"REQUEST_TIMEOUT"`
	Cycle              uint32 `yaml:"CYCLE"`
}

func (c *ServerConfig) GetConfig() *ServerConfig {
	yamlFile, err := ioutil.ReadFile("config/server.yaml") //这个路径相对于main函数文件的路径
	if err != nil {
		log.Println(err.Error())
	}

	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Println(err.Error())
	}

	return c
}

func GetId() string {
	value := int64(time.Now().Nanosecond()) << 32
	next := atomic.AddInt64(&value, 1)
	return strconv.FormatInt(next, 10)
}

func GetJobId(Handle, Params string) string {
	md5Ctx := md5.New()
	timeStr := strconv.FormatInt(time.Now().Unix(), 10)
	val := Handle + Params + timeStr
	//md5Ctx.Write([]byte(val))

	io.WriteString(md5Ctx, val)
	md5Str := md5Ctx.Sum(nil)

	return hex.EncodeToString(md5Str)
}

func GetBuffer(n int) (buf []byte) {
	buf = make([]byte, n)
	return
}

func IsMulParams(params []byte) bool {
	/*
		for _, v := range params {
			if v == PARAMS_SCOPE {
				return true
			}
		}

		return false
	*/

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
