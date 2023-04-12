package alert

import (
	"fmt"
	"time"

	"github.com/HughNian/nmid/pkg/model"
	"github.com/blinkbean/dingtalk"
)

var ding *dingtalk.DingTalk

var (
	DERROR   = "ERROR"
	DWARNING = "WARNING"
	DSUCCESS = "SUCCESS"
	DINFO    = "INFO"
)

var messageType = map[string]dingtalk.MarkType{
	"ERROR":   dingtalk.RED,
	"WARNING": dingtalk.GOLD,
	"SUCCESS": dingtalk.GREEN,
	"INFO":    dingtalk.BLUE,
}

func NewDingTalk(dingConfig *model.DingTalkConfig) *dingtalk.DingTalk {
	ding = dingtalk.InitDingTalkWithSecret(dingConfig.Tokens, dingConfig.Secret)
	return ding
}

func SendText(content string) {
	ding.SendTextMessage(content)
}

func SendTextAtAll(content string) {
	ding.SendTextMessage(content, dingtalk.WithAtAll())
}

func SendTextAtMobile(content string, mobiles []string) {
	ding.SendTextMessage(content, dingtalk.WithAtMobiles(mobiles))
}

func SendMarkDown(mtype, title, content string) {
	dm := dingtalk.DingMap()
	dm.Set(title, dingtalk.H2)
	dm.Set("---", dingtalk.N)
	dm.Set(mtype, messageType[mtype])
	ding.SendMarkDownMessageBySlice(title, dm.Slice())
}

func SendMarkDownAtAll(mtype, title, content string) {
	startTime := time.Now().Format("2006-01-02 15:04:05")

	dm := dingtalk.DingMap()
	dm.Set(title, dingtalk.H2)
	dm.Set("---", dingtalk.N)
	dm.Set(mtype, messageType[mtype])
	dm.Set(fmt.Sprintf("start time: %s", startTime), dingtalk.N)
	dm.Set("message: "+content, dingtalk.N)
	ding.SendMarkDownMessageBySlice(title, dm.Slice(), dingtalk.WithAtAll())
}

func SendMarkDownAtMobile(mtype, title, content string, mobiles []string) {
	startTime := time.Now().Format("2006-01-02 15:04:05")

	dm := dingtalk.DingMap()
	dm.Set(title, dingtalk.H2)
	dm.Set("---", dingtalk.N)
	dm.Set(mtype, messageType[mtype])
	dm.Set(fmt.Sprintf("start time: %s", startTime), dingtalk.N)
	dm.Set("message: "+content, dingtalk.N)
	ding.SendMarkDownMessageBySlice(title, dm.Slice(), dingtalk.WithAtMobiles(mobiles))
}
