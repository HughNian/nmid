package dashboard

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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
	Pid                                         int
	Arch                                        string
	HostName                                    string
	Os                                          string
	Osfamily                                    string
	Version                                     string
	GoVersion                                   string
	StartTime                                   time.Time
	UpTime                                      time.Duration
	DiscoveryNum                                int
	WorkerNum                                   int
	FuncNum                                     int
	SuccesNum                                   float64
	FailNum                                     float64
	CloseNum                                    float64
	WCurrentPage, FCurrentPage                  int
	WPageSize, FPageSize                        int
	WTotalPages, FTotalPages                    int
	IndexWorkerList, WorkerList, WorkersPerPage []metric.WorkerList
	IndexFuncList, FuncList, FuncsPerPage       []metric.FuncList

	Template string
}

func seq(a, b int) []int {
	var res []int
	for i := a; i <= b; i++ {
		res = append(res, i)
	}
	return res
}

func sub(a, b int) int {
	return a - b
}

func add(a, b int) int {
	return a + b
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
		GoVersion: runtime.Version(),
		StartTime: startTime,
		WPageSize: 10,
		FPageSize: 10,
	}

	return
}

func (d *Dashboard) index(w http.ResponseWriter, r *http.Request) {
	// 创建带自定义函数的模板
	tmpl := template.New("").Funcs(template.FuncMap{
		"sub": sub,
		"add": add,
		"seq": seq,
	})
	tmpl, err := tmpl.ParseFiles(d.Template)
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
	var iwlimit, iflimit = 5, 5
	if len(lists["workers"].([]metric.WorkerList)) < iwlimit {
		iwlimit = len(lists["workers"].([]metric.WorkerList))
	}
	if len(lists["funcs"].([]metric.FuncList)) < iflimit {
		iflimit = len(lists["funcs"].([]metric.FuncList))
	}
	d.IndexWorkerList = lists["workers"].([]metric.WorkerList)[0:iwlimit]
	d.IndexFuncList = lists["funcs"].([]metric.FuncList)[0:iflimit]

	d.UpTime = time.Since(d.StartTime)

	templateName := filepath.Base(d.Template)
	tmpl.ExecuteTemplate(w, templateName, d)
}

func (d *Dashboard) workers(w http.ResponseWriter, r *http.Request) {
	// 创建带自定义函数的模板
	tmpl := template.New("").Funcs(template.FuncMap{
		"sub": sub,
		"add": add,
		"seq": seq,
	})
	tmpl, err := tmpl.ParseFiles("dashboard/workers.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wpageStr := r.URL.Query().Get("wpage")
	if wpageStr == "" {
		d.WCurrentPage = 1
	} else {
		d.WCurrentPage, _ = strconv.Atoi(wpageStr)
	}

	lists := metric.GetWorkersFuncs()
	d.WorkerList = lists["workers"].([]metric.WorkerList)
	totalWorkers := len(d.WorkerList)
	d.WTotalPages = (totalWorkers + d.WPageSize - 1) / d.WPageSize // 向上取整
	start := (d.WCurrentPage - 1) * d.WPageSize
	end := start + d.WPageSize
	if end > totalWorkers {
		end = totalWorkers
	}
	d.WorkersPerPage = d.WorkerList[start:end]

	tmpl.ExecuteTemplate(w, "workers.html", d)
}

func (d *Dashboard) functions(w http.ResponseWriter, r *http.Request) {
	// 创建带自定义函数的模板
	tmpl := template.New("").Funcs(template.FuncMap{
		"sub": sub,
		"add": add,
		"seq": seq,
	})
	tmpl, err := tmpl.ParseFiles("dashboard/functions.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fpageStr := r.URL.Query().Get("fpage")
	if fpageStr == "" {
		d.FCurrentPage = 1
	} else {
		d.FCurrentPage, _ = strconv.Atoi(fpageStr)
	}

	lists := metric.GetWorkersFuncs()
	d.FuncList = lists["funcs"].([]metric.FuncList)
	totalFuncs := len(d.FuncList)
	d.FTotalPages = (totalFuncs + d.FPageSize - 1) / d.FPageSize
	startFunc := (d.FCurrentPage - 1) * d.FPageSize
	endFunc := startFunc + d.FPageSize
	if endFunc > totalFuncs {
		endFunc = totalFuncs
	}
	d.FuncsPerPage = d.FuncList[startFunc:endFunc]

	tmpl.ExecuteTemplate(w, "functions.html", d)
}

func (d *Dashboard) StartDashboard(c model.ServerConfig) {
	if len(c.Dashboard.Port) == 0 {
		return
	}

	once.Do(func() {
		thread.StartMinorGO("start dashboard server", func() {
			d.Template = c.Dashboard.Template

			http.HandleFunc(c.Dashboard.DefaultPath, d.index)
			http.HandleFunc("/workers", d.workers)
			http.HandleFunc("/functions", d.functions)

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
