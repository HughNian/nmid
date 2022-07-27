package security

import (
	"net"
	"nmid-v2/pkg/model"
)

func DoWhiteList(ip string, list *model.WhiteList) bool {
	if !list.Enable {
		return true
	}

	if list.AllowList[ip] {
		return true
	}

	remoteIP := net.ParseIP(ip)
	for _, mask := range list.AllowListMask {
		_, ipNet, err := net.ParseCIDR(mask)
		if nil == err {
			if ipNet.Contains(remoteIP) {
				return true
			}
		}
	}

	return false
}
