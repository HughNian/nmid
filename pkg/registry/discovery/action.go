package discovery

import "strings"

const (
	_ignorePrefix = "ignore.app"
)

var ignoreAppid = []string{_appid} // 需要精确过滤的appid

func (d *Discovery) GetAppList() []string {
	d.mutex.RLock()
	appIDS := []string{}
	i := 0
	for appID := range d.apps {
		if d.IsIgnored(appID) {
			continue
		}
		appIDS = append(appIDS, appID)
		i++
	}
	d.mutex.RUnlock()
	return appIDS
}

func (d *Discovery) IsIgnored(appid string) bool {
	for _, ignore := range ignoreAppid {
		if appid == ignore {
			return true
		}
	}
	return strings.HasPrefix(appid, _ignorePrefix)
}
