package thread

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/HughNian/nmid/pkg/logger"
)

var (
	gonum     int64 = 0    //main goroutine num manager
	goState   int64 = 0    //goroutine state 0-runing 1-need safe over
	needAlarm       = true //need send alarm message
)

// all goroutine is safe run
func IsGoRuntime() bool {
	return atomic.LoadInt64(&goState) == 0
}

// goroutine safe over
func GoSecurityOver() {
	atomic.StoreInt64(&goState, 1)
}

// start main goroutine & mark goroutine
func StartGo(mark string, f func(), overf func(isdebug bool)) {
	startGo(mark, true, f, overf)
}

// minor goroutine
// mark
// f main function
// overf over function
func StartMinorGO(mark string, f func(), overf func(isdebug bool)) {
	startGo(mark, false, f, overf)
}

// start goroutine
// mark
// ismain
// f
// overf
func startGo(mark string, ismain bool, f func(), overf func(isdebug bool)) {
	if f == nil {
		log.Panicln("start server fail:" + mark + ", f is nil")
	}
	if ismain {
		atomic.AddInt64(&gonum, 1)
	}
	go func() {
		logger.Infof("start go: %s, ismain: %t", mark, ismain)
		if ismain {
			log.Println("start go:", mark)
		}
		defer func() {
			isdebug := false
			if err := recover(); err != nil {
				logger.Debug(fmt.Sprint("[debug] ", mark, " error:", err, " stack:", string(debug.Stack())))
				isdebug = true
			}
			if ismain {
				log.Println("end go:", mark, ",isdebug:", isdebug)
			}
			logger.Infof("server over mark: %v ,ismain: %t", mark, ismain)
			if overf != nil {
				func() {
					defer func() {
						ListenDebug(mark + " overf bug")
					}()
					overf(isdebug)
				}()
			}
			if ismain {
				atomic.AddInt64(&gonum, -1)
				GoSecurityOver()
			}
		}()
		f()
	}()
}

func ListenDebug(mark string) bool {
	if err := recover(); err != nil {
		logger.Debug("[debug] %s  error: %s stack: %s", mark, err, string(debug.Stack()))
		return true
	}
	return false
}

func ListenAllGO(stopall func(), alarmGroup string, alarmContent string) {
	ListenKill()
	flag := false
	for {
		time.Sleep(2 * time.Second)
		if !flag && !IsGoRuntime() {
			flag = true
			if stopall != nil {
				stopall()
			}
			logger.Infof("stopall goruntime")
		}
		v := atomic.LoadInt64(&gonum)
		if v <= 0 {
			if needAlarm {
				//todo send alarm
			}

			logger.Infof("all go over")
			return
		}
	}
}

func ListenKill() {
	StartMinorGO("listen kill", func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		s := <-c

		needAlarm = false
		logger.Infof("Server Exit: %s", s.String())
		atomic.StoreInt64(&goState, 1)
	}, nil)
}
