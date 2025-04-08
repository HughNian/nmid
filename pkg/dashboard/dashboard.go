package dashboard

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"text/template"
	"time"

	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/metric"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/thread"
)

var (
	once sync.Once
)

type Dashboard struct {
	Arch         string
	HostName     string
	Os           string
	Osfamily     string
	Pid          int
	Version      string
	StartTime    time.Time
	UpTime       time.Duration
	DiscoveryNum int
	WorkerNum    int
	FuncNum      int
	SuccesNum    float64
	FailNum      float64
	CloseNum     float64
	WorkerList   map[string]metric.WorkerList
	FuncList     map[string]metric.FuncList

	Template string
}

func NewDashboard(startTime time.Time, version string) (d *Dashboard) {
	hostName, _ := os.Hostname()

	d = &Dashboard{
		Arch:      runtime.GOARCH,
		HostName:  hostName,
		Os:        runtime.GOOS,
		Osfamily:  getOSFamily(),
		Pid:       os.Getpid(),
		Version:   version,
		StartTime: startTime,
	}

	return
}

func (d *Dashboard) handler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(d.Template)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dwfNum := metric.GetDiscoveryWorkerFuncNum()
	d.DiscoveryNum = dwfNum["discovery_num"]
	d.WorkerNum = dwfNum["worker_num"]
	d.FuncNum = dwfNum["func_num"]
	sfcNum := metric.GetSuccesFailCloseNum()
	d.SuccesNum = sfcNum["success_num"]
	d.FailNum = sfcNum["fail_num"]
	d.CloseNum = sfcNum["close_num"]
	lists := metric.GetWorkersFuncs()
	d.WorkerList = lists["workers"].(map[string]metric.WorkerList)
	d.FuncList = lists["funcs"].(map[string]metric.FuncList)

	d.UpTime = time.Since(d.StartTime)

	tmpl.Execute(w, d)
}

func (d *Dashboard) StartDashboard(c model.ServerConfig) {
	if len(c.Dashboard.Port) == 0 {
		return
	}

	once.Do(func() {
		thread.StartMinorGO("start dashboard server", func() {
			d.Template = c.Dashboard.Template

			http.HandleFunc(c.Dashboard.DefaultPath, d.handler)

			dashboradAddr := fmt.Sprintf("%s:%s", c.Dashboard.Host, c.Dashboard.Port)
			logger.Infof("starting dashboard server at %s", dashboradAddr)

			if err := http.ListenAndServe(dashboradAddr, nil); err != nil {
				logger.Error(err)
			}
		}, func(isdebug bool) {
			fmt.Println("dashboard server over")
		})
	})
}
