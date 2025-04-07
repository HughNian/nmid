package dashboard

import (
	"os"
	"runtime"
	"time"
)

type PageData struct {
	Pid      int
	Os       string
	Arch     string
	Osfamily string
	HostName string
	Version  string
	UpTime   time.Duration
}

func getOSFamily() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return "Unix"
	}
}

// 获取基础信息
func GetBasicHostInfo() (pdata PageData) {
	//主机名
	pdata.HostName, _ = os.Hostname()
	pdata.Version = runtime.Version()

	//操作系统信息
	pdata.Os = runtime.GOOS
	pdata.Arch = runtime.GOARCH
	pdata.Osfamily = getOSFamily()

	//进程信息
	pdata.Pid = os.Getpid()
	pdata.UpTime = time.Since(time.Now())

	return
}
