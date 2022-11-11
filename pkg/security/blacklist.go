package security

import (
	"net"
	"nmid/pkg/model"
)

func DoBlackList(ip string, list *model.BlackList) bool {
	if !list.Enable {
		return false
	}

	if list.NoAllowList[ip] {
		return false
	}

	remoteIP := net.ParseIP(ip)
	for _, mask := range list.NoAllowListMask {
		_, ipNet, err := net.ParseCIDR(mask)
		if nil == err {
			if ipNet.Contains(remoteIP) {
				return false
			}
		}
	}

	return true
}
