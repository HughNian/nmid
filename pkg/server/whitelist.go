package server

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net"
)

type WhiteList struct {
	Enable        bool            `yaml:"ENABLE"`
	AllowList     map[string]bool `yaml:"ALLOWLIST"`
	AllowListMask []*net.IPNet    `yaml:"ALLOWLISTMASK"` //net.ParseCIDR("172.17.0.0/16") to get *net.IPNet
}

func GetWhiteList() *WhiteList {
	var wh *WhiteList

	listFile, err := ioutil.ReadFile("config/whitelist.yaml")
	if err != nil {
		log.Println(err.Error())
	}

	err = yaml.Unmarshal(listFile, wh)
	if err != nil {
		log.Println(err.Error())
	}

	return wh
}

func (wh *WhiteList) DoWhiteList(ip string) bool {
	if !wh.Enable {
		return true
	}

	if wh.AllowList[ip] {
		return true
	}

	remoteIP := net.ParseIP(ip)
	for _, mask := range wh.AllowListMask {
		if mask.Contains(remoteIP) {
			return true
		}
	}

	return false
}
