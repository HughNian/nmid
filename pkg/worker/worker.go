package worker

import (
	"errors"
	"fmt"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/HughNian/nmid/pkg/trace"
	"github.com/HughNian/nmid/pkg/utils"
	"github.com/SkyAPM/go2sky"
	"sync"
	"time"
)

//rpc tcp worker

type Worker struct {
	sync.Mutex

	WorkerId   string
	WorkerName string

	Agents   []*Agent
	Funcs    map[string]*Function
	FuncsNum int
	Resps    chan *Response

	Reporter go2sky.Reporter
	Tracer   *go2sky.Tracer

	ready    bool
	running  bool
	useTrace bool
}

func NewWorker() *Worker {
	wor := &Worker{
		Agents:   make([]*Agent, 0),
		Funcs:    make(map[string]*Function),
		FuncsNum: 0,
		Resps:    make(chan *Response, model.QUEUE_SIZE),
		ready:    false,
		running:  false,
		useTrace: false,
	}

	return wor
}

func (w *Worker) SetWorkerId(wid string) *Worker {
	if len(wid) == 0 {
		w.WorkerId = utils.GetId()
	} else {
		w.WorkerId = wid
	}
	return w
}

func (w *Worker) SetWorkerName(wname string) *Worker {
	if len(wname) == 0 {
		w.WorkerName = utils.GetId()
	} else {
		w.WorkerName = wname
	}

	return w
}

func (w *Worker) WithTrace(reporterUrl string) *Worker {
	w.useTrace = true
	w.Reporter, w.Tracer = trace.NewReporter(reporterUrl, w.WorkerName)
	return w
}

func (w *Worker) AddServer(net, addr string) (err error) {
	var agent *Agent = NewAgent(net, addr, w)

	if agent == nil {
		return fmt.Errorf("agent nil")
	}
	w.Agents = append(w.Agents, agent)

	return nil
}

func (w *Worker) AddFunction(funcName string, jobFunc JobFunc) (err error) {
	w.Lock()
	if _, ok := w.Funcs[funcName]; ok {
		return fmt.Errorf("function already exist")
	}

	w.Funcs[funcName] = NewFunction(jobFunc, funcName)
	w.FuncsNum++
	w.Unlock()

	if w.running {
		go w.FuncBroadcast(funcName, model.PDT_W_ADD_FUNC)
	}

	return nil
}

func (w *Worker) DelFunction(funcName string) (err error) {
	w.Lock()
	if _, ok := w.Funcs[funcName]; !ok {
		return fmt.Errorf("function not exist")
	}

	delete(w.Funcs, funcName)
	w.FuncsNum--
	w.Unlock()

	if w.running {
		go w.FuncBroadcast(funcName, model.PDT_W_DEL_FUNC)
	}

	return nil
}

func (w *Worker) GetFunction(funcName string) (function *Function, err error) {
	if len(w.Funcs) == 0 || w.FuncsNum == 0 {
		return nil, fmt.Errorf("worker have no funcs")
	}

	w.Lock()
	f, ok := w.Funcs[funcName]
	w.Unlock()

	if f == nil || !ok {
		return nil, fmt.Errorf("not found")
	}

	if f.FuncName != funcName {
		return nil, fmt.Errorf("not found")
	}

	function = f

	return function, nil
}

func (w *Worker) DoFunction(resp *Response) (err error) {
	if resp.DataType == model.PDT_S_GET_DATA {
		//use trace
		if w.useTrace {
			//set entry span
			resp.SetEntrySpan()
		}

		funcName := resp.Handle
		if function, err := w.GetFunction(funcName); err != nil {
			return err
		} else if function != nil {
			if function.FuncName != funcName {
				return fmt.Errorf("funcname error")
			}
			if resp.ParamsLen == 0 {
				return fmt.Errorf("params error")
			}

			var ret []byte
			if ret, err = function.Func(resp); err == nil {
				resp.Agent.Req.HandleLen = resp.HandleLen
				resp.Agent.Req.Handle = resp.Handle
				resp.Agent.Req.ParamsLen = resp.ParamsLen
				resp.Agent.Req.Params = resp.Params
				resp.Agent.Req.JobIdLen = resp.JobIdLen
				resp.Agent.Req.JobId = resp.JobId

				resp.Agent.Lock()
				resp.Agent.Req.RetPack(ret)
				resp.Agent.Write()
				resp.Agent.Unlock()
			}
		}
	}

	return nil
}

func (w *Worker) FuncBroadcast(funcName string, flag int) {
	for _, a := range w.Agents {
		switch flag {
		case model.PDT_W_ADD_FUNC:
			a.Req.AddFunctionPack(funcName)
		case model.PDT_W_DEL_FUNC:
			a.Req.DelFunctionPack(funcName)
		default:
			a.Req.AddFunctionPack(funcName)
		}
		a.Write()
	}
}

func (w *Worker) WorkerReady() (err error) {
	if len(w.Agents) == 0 {
		return errors.New("none active agents")
	}
	if w.FuncsNum == 0 || len(w.Funcs) == 0 {
		return errors.New("none functions")
	}

	for _, a := range w.Agents {
		if err = a.Connect(); err != nil {
			return err
		}
	}

	for fn := range w.Funcs {
		w.FuncBroadcast(fn, model.PDT_W_ADD_FUNC)
	}

	w.Lock()
	w.ready = true
	w.Unlock()

	return nil
}

func (w *Worker) WorkerDo() {
	if !w.ready {
		err := w.WorkerReady()
		if err != nil {
			logger.Fatal(err)
		}
	}

	w.Lock()
	w.running = true
	w.Unlock()

	for _, a := range w.Agents {
		go a.Grab()
	}

	for resp := range w.Resps {
		switch resp.DataType {
		case model.PDT_TOSLEEP:
			time.Sleep(time.Duration(2) * time.Second)
			go resp.Agent.Wakeup()

			//fallthrough
		case model.PDT_S_GET_DATA:
			if err := w.DoFunction(resp); err != nil {
				logger.Error(err)
			}
			//fallthrough
		case model.PDT_NO_JOB:
			go resp.Agent.Grab()

		case model.PDT_WAKEUPED:
		default:
			go resp.Agent.Grab()
		}
	}
}

func (w *Worker) WorkerClose() {
	if w.running {
		for _, a := range w.Agents {
			a.Close()
		}

		w.running = false
		close(w.Resps)

		if w.useTrace {
			w.Reporter.Close()
		}
	}
}
