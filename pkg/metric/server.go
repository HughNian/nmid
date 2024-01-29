package metric

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/thread"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	once           sync.Once
	closeListeners = new(closeManager)
)

type closeManager struct {
	lock      sync.Mutex
	waitGroup sync.WaitGroup
	listeners []func()
}

func (cm *closeManager) addListener(fn func()) (waitCalled func()) {
	cm.waitGroup.Add(1)

	cm.lock.Lock()
	cm.listeners = append(cm.listeners, func() {
		defer cm.waitGroup.Done()
		fn()
	})
	cm.lock.Unlock()

	return func() {
		cm.waitGroup.Wait()
	}
}

func (cm *closeManager) doListeners() {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	fmt.Println(len(cm.listeners))

	for _, listener := range cm.listeners {
		if listener != nil {
			thread.StartMinorGO("close prometheus metric listeners", listener, nil)
		}
	}
}

func AddCloseListener(fn func()) (waitCalled func()) {
	return closeListeners.addListener(fn)
}

func DoCloseListener() {
	closeListeners.doListeners()
}

func DoCloseListenerWithWg(wg *sync.WaitGroup) {
	defer wg.Done()

	DoCloseListener()
}

// starts prometheus.
func StartServer(c model.ServerConfig) {
	if len(c.Prometheus.Port) == 0 {
		return
	}

	once.Do(func() {
		thread.StartMinorGO("start prometheus server", func() {
			http.Handle(c.Prometheus.Path, promhttp.Handler())

			promeAddr := fmt.Sprintf("%s:%s", c.Prometheus.Host, c.Prometheus.Port)
			logger.Infof("starting prometheus server at %s", promeAddr)

			if err := http.ListenAndServe(promeAddr, nil); err != nil {
				logger.Error(err)
			}
		}, func(isdebug bool) {
			fmt.Println("prometheus server over")
		})
	})
}
