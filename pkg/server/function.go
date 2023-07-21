package server

import (
	"math/rand"
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/model"
	wr "github.com/mroth/weightedrand"
)

type Func struct {
	FuncName string

	WorkerNum      int
	LoadBlanceType int
	Workers        []*SWorker
}

type FuncMap struct {
	FuncNum int
	Funcs   sync.Map

	mutex sync.Mutex
}

func NewFuncMap() *FuncMap {
	return &FuncMap{
		FuncNum: 0,
	}
}

func (fm *FuncMap) AddFunc(worker *SWorker, name string) bool {
	if worker == nil {
		return false
	}

	if len(name) == 0 {
		return false
	}

	var function *Func
	if item, exist := fm.Funcs.Load(name); exist {
		function = item.(*Func)
	} else {
		function = &Func{
			FuncName:       name,
			LoadBlanceType: model.LOADBLANCE_HASH,
			Workers:        make([]*SWorker, 0),
		}

		fm.mutex.Lock()
		fm.FuncNum++
		fm.mutex.Unlock()
		fm.Funcs.Store(name, function)
	}

	fm.mutex.Lock()
	function.WorkerNum++
	fm.mutex.Unlock()
	function.Workers = append(function.Workers, worker)

	return true
}

func (fm *FuncMap) GetFunc(name string) *Func {
	if item, exist := fm.Funcs.Load(name); exist {
		function := item.(*Func)
		return function
	}

	return nil
}

func (fm *FuncMap) DelAllFunc(name string) bool {
	if _, exist := fm.Funcs.Load(name); exist {
		fm.Funcs.Delete(name)
		fm.mutex.Lock()
		fm.FuncNum--
		fm.mutex.Unlock()
		return true
	}

	return false
}

func (fm *FuncMap) DelWorkerFunc(workerId, name string) bool {
	if len(name) == 0 {
		return false
	}

	function := fm.GetFunc(name)
	if function == nil {
		return false
	}
	if function.WorkerNum == 0 {
		return false
	}
	if function.FuncName != name {
		return false
	}

	var worker *SWorker
	worker = nil
	for k, w := range function.Workers {
		if w.WorkerId == workerId {
			worker = w
			function.Workers = append(function.Workers[:k], function.Workers[k+1:]...)
			fm.mutex.Lock()
			function.WorkerNum--
			fm.mutex.Unlock()
			break
		}
	}
	if worker == nil {
		return false
	}

	if function.WorkerNum == 0 {
		fm.DelAllFunc(name)
	}
	return true
}

func (fm *FuncMap) CleanWorkerFunc(workerId string) bool {
	fm.Funcs.Range(func(key, item interface{}) bool {
		function := item.(*Func)
		fm.DelWorkerFunc(workerId, function.FuncName)
		return true
	})

	return true
}

func (fm *FuncMap) GetBestWorker(name string) (worker *SWorker) {
	if item, exist := fm.Funcs.Load(name); exist {
		function := item.(*Func)
		if function.WorkerNum > 0 {
			var best *SWorker

			var hashfunc = func() *SWorker {
				rkey := int(rand.Int() % function.WorkerNum)
				return function.Workers[rkey]
			}

			switch function.LoadBlanceType {
			//一致性hash
			case model.LOADBLANCE_HASH:
				best = hashfunc()

			//加权随机
			case model.LOADBLANCE_ROUND_WEIGHT:
				{
					if function.WorkerNum <= 10 {
						rand.Seed(time.Now().UTC().UnixNano())

						ch := make([]wr.Choice, 0)
						for _, w := range function.Workers {
							ch = append(ch, wr.NewChoice(w, w.Weight))
						}

						chooser, err := wr.NewChooser(ch...)
						if nil == err {
							best = chooser.Pick().(*SWorker)
						}

						//以防止weight都为0的情况
						if nil == best {
							best = hashfunc()
						}
					} else {
						best = hashfunc()
					}
				}

			//lru 最少使用率
			case model.LOADBLANCE_LRU:
				for _, val := range function.Workers {
					if val.Jobs.GetJobNum() < best.Jobs.GetJobNum() {
						best = val
					}
				}

			//默认，一致性hash
			default:
				best = hashfunc()
			}

			worker = best
			return worker
		}

		return
	}

	return
}

func (fm *FuncMap) DelWorker(workerId string) bool {
	var ret bool
	if fm.FuncNum == 0 {
		return true
	}

	fm.Funcs.Range(func(key, val interface{}) bool {
		function, ok := val.(*Func)
		if !ok {
			ret = false
			return ret
		}

		for k, worker := range function.Workers {
			if worker.WorkerId == workerId {
				function.Workers = append(function.Workers[:k], function.Workers[k+1:]...)
				fm.mutex.Lock()
				function.WorkerNum--
				fm.mutex.Unlock()
				break
			}
		}

		ret = true
		return ret
	})

	return ret
}
