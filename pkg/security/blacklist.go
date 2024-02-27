package security

import (
	"net"

	"github.com/HughNian/nmid/pkg/conf"
)

func DoBlackList(ip string) bool {
	if !conf.GetConfig().BlackList.Enable {
		return false
	}

	if conf.GetConfig().BlackList.NoAllowList[ip] {
		return false
	}

	remoteIP := net.ParseIP(ip)
	for _, mask := range conf.GetConfig().BlackList.NoAllowListMask {
		_, ipNet, err := net.ParseCIDR(mask)
		if nil == err {
			if ipNet.Contains(remoteIP) {
				return false
			}
		}
	}

	return true
}
