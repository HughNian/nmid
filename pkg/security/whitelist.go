package security

import (
	"net"

	"github.com/HughNian/nmid/pkg/conf"
)

func DoWhiteList(ip string) bool {
	if !conf.GetConfig().WhiteList.Enable {
		return true
	}

	if conf.GetConfig().WhiteList.AllowList[ip] {
		return true
	}

	remoteIP := net.ParseIP(ip)
	for _, mask := range conf.GetConfig().WhiteList.AllowListMask {
		_, ipNet, err := net.ParseCIDR(mask)
		if nil == err {
			if ipNet.Contains(remoteIP) {
				return true
			}
		}
	}

	return false
}
