package alert

import (
	"fmt"
	"os"
	"time"

	"github.com/blinkbean/dingtalk"
	"github.com/joho/godotenv"
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

func init() {
	godotenv.Load("./.env")

	if os.Getenv("DINGTAKL_ENABLE") == "true" {
		NewDingTalk(os.Getenv("DINGTAKL_TOKENS"), os.Getenv("DINGTAKL_SECRET"))
	}
}

func NewDingTalk(tokens, secret string) *dingtalk.DingTalk {
	ding = dingtalk.InitDingTalkWithSecret(tokens, secret)
	return ding
}

func SendText(content string) {
	if nil == ding {
		return
	}

	ding.SendTextMessage(content)
}

func SendTextAtAll(content string) {
	if nil == ding {
		return
	}

	ding.SendTextMessage(content, dingtalk.WithAtAll())
}

func SendTextAtMobile(content string, mobiles []string) {
	if nil == ding {
		return
	}

	ding.SendTextMessage(content, dingtalk.WithAtMobiles(mobiles))
}

func SendMarkDown(mtype, title, content string) {
	if nil == ding {
		return
	}

	dm := dingtalk.DingMap()
	dm.Set(title, dingtalk.H2)
	dm.Set("---", dingtalk.N)
	dm.Set(mtype, messageType[mtype])
	ding.SendMarkDownMessageBySlice(title, dm.Slice())
}

func SendMarkDownAtAll(mtype, title, content string) {
	if nil == ding {
		return
	}

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
	if nil == ding {
		return
	}

	startTime := time.Now().Format("2006-01-02 15:04:05")

	dm := dingtalk.DingMap()
	dm.Set(title, dingtalk.H2)
	dm.Set("---", dingtalk.N)
	dm.Set(mtype, messageType[mtype])
	dm.Set(fmt.Sprintf("start time: %s", startTime), dingtalk.N)
	dm.Set("message: "+content, dingtalk.N)
	ding.SendMarkDownMessageBySlice(title, dm.Slice(), dingtalk.WithAtMobiles(mobiles))
}
