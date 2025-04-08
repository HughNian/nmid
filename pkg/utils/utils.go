package utils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vmihailenco/msgpack"
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

func getMilliSec() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func GetNowSecond() int64 {
	milliSec := getMilliSec()
	return atomic.LoadInt64(&milliSec)
}

type IpInfo struct {
	Status      string  `json:"status"`
	Country     string  `json:"Country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	Isp         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
	Query       string  `json:"query"`
}

func GetIPInfo(ip string) (info []byte) {
	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	info = body

	return
}

type LocationInfo struct {
	Status    int `json:"status"`
	Message   int `json:"message"`
	RequestId int `json:"request_id"`
	Result    struct {
		Address            string `json:"address"`
		FormattedAddresses struct {
			Recommend string `json:"recommend"`
			Rough     string `json:"rough"`
		} `json:"formatted_addresses"`
	} `json:"result"`
}

func GetLocation(lat, lon float64) *LocationInfo {
	var loca = &LocationInfo{}
	mapkey := os.Getenv("QQ_LBS_KEY")

	url := fmt.Sprintf("https://apis.map.qq.com/ws/geocoder/v1/?location=%.4f,%.4f&key=%s&get_poi=1", lat, lon, mapkey)
	// fmt.Println("location url", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error:", err)
		return loca
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error:", err)
		return loca
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println("Error:", err)
		return loca
	}

	// fmt.Println("loca ret", string(body))

	json.Unmarshal(body, loca)

	return loca
}

type ZoneInfo struct {
	Ip      string
	Zone    string
	Country string
	Prov    string
	City    string
	Lat     string
	Lon     string
}

func GetIPZone(ip string) (zinfo ZoneInfo) {
	ret := GetIPInfo(ip)
	if len(ret) > 0 {
		var info IpInfo
		err := json.Unmarshal(ret, &info)
		if nil == err {
			if info.Status == "success" {
				location := GetLocation(info.Lat, info.Lon)
				var address string
				if location.Status == 0 {
					address = fmt.Sprintf("%s, %s", location.Result.Address, location.Result.FormattedAddresses.Recommend)
				}

				zinfo.Ip = ip
				zinfo.Zone = fmt.Sprintf("%s-%s-%s-%s", info.Country, info.City, info.Org, address)
				zinfo.Lat = fmt.Sprintf("%.3f", info.Lat)
				zinfo.Lon = fmt.Sprintf("%.3f", info.Lon)
				zinfo.Country = info.Country
				zinfo.Prov = info.RegionName
				zinfo.City = info.City
			}
		}
	}

	return
}

func RandomInt(min, max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max-min+1) + min
}

// 获取服务器第一个有效 IPv4 地址（排除回环和 Docker 虚拟接口）
func GetServerIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("get interfaces failed: %v", err)
	}

	for _, iface := range interfaces {
		// 排除 Docker 虚拟接口
		if strings.Contains(iface.Name, "docker") ||
			strings.Contains(iface.Name, "lo") {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.IsLoopback() {
				continue
			}

			if ipv4 := ipNet.IP.To4(); ipv4 != nil {
				return ipv4.String(), nil
			}
		}
	}

	return "", errors.New("no valid IPv4 address found")
}
