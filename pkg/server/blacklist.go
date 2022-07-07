package server

import (
	"net"
	"nmid-v2/pkg/conf"
)

func DoBlackList(ip string, list *conf.BlackList) bool {
	if !list.Enable {
		return false
	}

	if list.NoAllowList[ip] {
		return false
	}

	remoteIP := net.ParseIP(ip)
	for _, mask := range list.NoAllowListMask {
		if mask.Contains(remoteIP) {
			return false
		}
	}

	return true
}
