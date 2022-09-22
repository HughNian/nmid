package nmidr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/pkg/ecode"
	http "github.com/go-kratos/kratos/pkg/net/http/blademaster"
	xtime "github.com/go-kratos/kratos/pkg/time"
	"math/rand"
	"net/url"
	"nmid-v2/pkg/logger"
	"nmid-v2/pkg/registry"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	_registerURL = "http://%s/registry/register"
	_logoffURL   = "http://%s/registry/logoff"
	_watchURL    = "http://%s/registry/watch"
	_renewURL    = "http://%s/registry/renew"
	_fetchAllURL = "http://%s/registry/fetch/all"

	_renewGap = 30 * time.Second
)

//Nmidr nmid-registry client
type Nmidr struct {
	conf       *registry.Config
	once       sync.Once
	ctx        context.Context
	cancelFunc context.CancelFunc
	mutex      sync.RWMutex
	httpClient *http.Client

	services map[string]*serviceInfo //serviceid -> *serviceInfo
	registry map[string]struct{}
}

type serviceInfo struct {
	resolver  map[string]Resolve //serviceid -> Resolve
	instances atomic.Value       //instances info
	lastTs    int64              // latest timestamp
}

type Resolve struct {
	nr    *Nmidr
	sid   string
	event chan struct{}
}

func (r *Resolve) Fetch() (info *registry.InstancesInfo, ret bool) {
	if _, ok := <-r.event; ok {
		ctx, cancel := context.WithCancel(r.nr.ctx)
		insInfo, err := r.nr.doFetchAll(ctx, r.sid) //fetch one serviceid instance info
		if nil != err {
			cancel()

			ret = false
			return
		}

		info = &registry.InstancesInfo{
			Instances: make(map[string][]*registry.Instance),
		}

		if info, ok := insInfo[r.sid]; ok {
			r.nr.mutex.RLock()
			service, ok := r.nr.services[r.sid]
			r.nr.mutex.RUnlock()
			if ok {
				service.lastTs = info.LastTs
				service.instances.Store(info)
			}
		}

		ret = true
		return
	}

	ret = true

	return info, ret
}

func (r *Resolve) Watch() <-chan struct{} {
	ctx, cancel := context.WithCancel(r.nr.ctx)
	data, err := r.nr.doWatch(ctx, r.sid)
	if nil != err {
		cancel()
	}

	if (data.WType == 0 || data.WType == 1) && data.WKey == r.sid {
		r.event <- struct{}{}
	}

	return r.event
}

func (r *Resolve) Close() error {
	r.nr.mutex.Lock()
	if service, ok := r.nr.services[r.sid]; ok && len(service.resolver) != 0 {
		delete(service.resolver, r.sid)
	}
	r.nr.mutex.Unlock()
	return nil
}

// watch every 30 minutes watch etcd
func (sci *serviceInfo) watch(serviceId string) {
	if resolve, ok := sci.resolver[serviceId]; ok {
		ticker := time.NewTicker(time.Minute * 30)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				resolve.Watch()
			}
		}
	}
}

// NewRegistry new a nmidr client
func NewRegistry(conf *registry.Config) (nr *Nmidr) {
	ctx, cancel := context.WithCancel(context.Background())
	nr = &Nmidr{
		conf:       conf,
		ctx:        ctx,
		cancelFunc: cancel,
		registry:   map[string]struct{}{},
	}
	// httpClient
	cfg := &http.ClientConfig{
		Dial:      xtime.Duration(3 * time.Second),
		KeepAlive: xtime.Duration(40 * time.Second),
		Timeout:   xtime.Duration(40 * time.Second),
	}
	nr.httpClient = http.NewClient(cfg)

	return
}

func (nr *Nmidr) Build(serviceId string) registry.Resolver {
	r := &Resolve{
		nr:    nr,
		sid:   serviceId,
		event: make(chan struct{}, 1),
	}
	nr.mutex.Lock()
	service, ok := nr.services[serviceId]
	if !ok {
		service = &serviceInfo{
			resolver: make(map[string]Resolve),
		}
		nr.services[serviceId] = service
	}
	service.resolver[r.sid] = *r
	nr.mutex.Unlock()

	if ok {
		select {
		case r.event <- struct{}{}:
		default:
		}
	}

	nr.once.Do(func() {
		go service.watch(serviceId)
		logger.Info("nmidr addWatch(%s) already watch(%v)", serviceId, ok)
	})

	return r
}

func (nr *Nmidr) pickMasterNode() string {
	return nr.conf.Nodes[rand.Intn(len(nr.conf.Nodes))]
}

func (nr *Nmidr) newParams(c *registry.Config) url.Values {
	params := url.Values{}
	params.Set("region", c.Region)
	params.Set("zone", c.Zone)
	params.Set("env", c.Env)
	params.Set("hostname", c.Host)
	return params
}

func (nr *Nmidr) Register(ins *registry.Instance) (cancelFunc context.CancelFunc, err error) {
	nr.mutex.Lock()
	if _, ok := nr.registry[ins.ServiceId]; ok {
		err = errors.New("instance duplicate registration")
	} else {
		nr.registry[ins.ServiceId] = struct{}{}
	}
	nr.mutex.Unlock()
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(nr.ctx)
	err = nr.doRegister(ctx, ins)
	if err != nil {
		nr.mutex.Lock()
		delete(nr.registry, ins.ServiceId)
		nr.mutex.Unlock()
		cancel()
		return
	}
	cancelFunc = context.CancelFunc(func() {
		cancel()
	})

	go func() {
		ticker := time.NewTicker(_renewGap)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := nr.doRenew(ctx, ins); err != nil && ecode.EqualError(ecode.NothingFound, err) {
					_ = nr.doRegister(ctx, ins)
				}
			case <-ctx.Done():
				_ = nr.doLogOff(ins)
				return
			}
		}
	}()

	return
}

func (nr *Nmidr) Close() error {
	nr.cancelFunc()
	return nil
}

func (nr *Nmidr) doRegister(ctx context.Context, ins *registry.Instance) (err error) {
	nr.mutex.RLock()
	conf := nr.conf
	nr.mutex.RUnlock()

	var metadata []byte
	if ins.Metadata != nil {
		if metadata, err = json.Marshal(ins.Metadata); err != nil {
			logger.Error("nmidr register instance Marshal metadata(%v) failed!error(%v)", ins.Metadata, err)
		}
	}

	uri := fmt.Sprintf(_registerURL, nr.pickMasterNode())
	params := nr.newParams(conf)
	params.Set("inflow_addr", ins.InFlowAddr)
	params.Set("outflow_addr", ins.OutFlowAddr)
	params.Set("service_id", ins.ServiceId)
	for _, addr := range ins.Addrs {
		params.Add("addrs", addr)
	}
	params.Set("version", ins.Version)
	params.Set("metadata", string(metadata))
	if ins.Status == 0 {
		params.Set("status", "1")
	} else {
		params.Set("status", strconv.FormatInt(ins.Status, 10))
	}

	res := new(struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	})
	if err = nr.httpClient.Post(ctx, uri, "", params, &res); err != nil {
		logger.Error("nmidr: register client.Get(%v)  zone(%s) env(%s) service_id(%s) addrs(%v) error(%v)",
			uri, conf.Zone, conf.Env, ins.ServiceId, ins.Addrs, err)
		return
	}
	if ec := ecode.Int(res.Code); !ecode.EqualError(ecode.OK, ec) {
		logger.Warn("nmidr register client.Get(%v)  env(%s) service_id(%s) addrs(%v) code(%v)", uri, conf.Env, ins.ServiceId, ins.Addrs, res.Code)
		err = ec
		return
	}
	logger.Info("nmidr register client.Get(%v) env(%s) service_id(%s) addrs(%s) success", uri, conf.Env, ins.ServiceId, ins.Addrs)

	return
}

func (nr *Nmidr) doRenew(ctx context.Context, ins *registry.Instance) (err error) {
	nr.mutex.RLock()
	c := nr.conf
	nr.mutex.RUnlock()

	uri := fmt.Sprintf(_renewURL, nr.pickMasterNode())
	params := nr.newParams(c)
	params.Set("inflow_addr", ins.InFlowAddr)
	params.Set("outflow_addr", ins.OutFlowAddr)
	params.Set("service_id", ins.ServiceId)

	res := new(struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	})
	if err = nr.httpClient.Post(ctx, uri, "", params, &res); err != nil {
		logger.Error("nmidr renew client.Get(%v)  env(%s) service_id(%s) hostname(%s) error(%v)",
			uri, c.Env, ins.ServiceId, c.Host, err)
		return
	}
	if ec := ecode.Int(res.Code); !ecode.EqualError(ecode.OK, ec) {
		err = ec
		if ecode.EqualError(ecode.NothingFound, ec) {
			return
		}
		logger.Error("nmidr renew client.Get(%v) env(%s) service_id(%s) hostname(%s) code(%v)",
			uri, c.Env, ins.ServiceId, c.Host, res.Code)
		return
	}

	return
}

func (nr *Nmidr) doLogOff(ins *registry.Instance) (err error) {
	nr.mutex.RLock()
	c := nr.conf
	nr.mutex.RUnlock()

	uri := fmt.Sprintf(_logoffURL, nr.pickMasterNode())
	params := nr.newParams(c)
	params.Set("inflow_addr", ins.InFlowAddr)
	params.Set("outflow_addr", ins.OutFlowAddr)
	params.Set("service_id", ins.ServiceId)

	res := new(struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	})
	if err = nr.httpClient.Post(context.TODO(), uri, "", params, &res); err != nil {
		logger.Error("nmidr cancel client.Get(%v) env(%s) service_id(%s) hostname(%s) error(%v)",
			uri, c.Env, ins.ServiceId, c.Host, err)
		return
	}
	if ec := ecode.Int(res.Code); !ecode.EqualError(ecode.OK, ec) {
		logger.Warn("nmidr cancel client.Get(%v)  env(%s) service_id(%s) hostname(%s) code(%v)",
			uri, c.Env, ins.ServiceId, c.Host, res.Code)
		err = ec
		return
	}
	logger.Info("nmidr cancel client.Get(%v)  env(%s) service_id(%s) hostname(%s) success",
		uri, c.Env, ins.ServiceId, c.Host)

	return
}

func (nr *Nmidr) doFetchAll(ctx context.Context, ServiceId string) (data map[string]*registry.InstancesInfo, err error) {
	nr.mutex.RLock()
	c := nr.conf
	nr.mutex.RUnlock()

	uri := fmt.Sprintf(_fetchAllURL, nr.pickMasterNode())
	params := nr.newParams(c)
	params.Set("service_id", ServiceId)

	res := new(struct {
		Code    int                                `json:"code"`
		Message string                             `json:"message"`
		Data    map[string]*registry.InstancesInfo `json:"data"`
	})
	if err = nr.httpClient.Post(ctx, uri, "", params, &res); err != nil {
		logger.Error("nmidr cancel client.Get(%v) env(%s) service_id(%s) hostname(%s) error(%v)",
			uri, c.Env, ServiceId, c.Host, err)
		return
	}
	if ec := ecode.Int(res.Code); !ecode.EqualError(ecode.OK, ec) {
		logger.Warn("nmidr cancel client.Get(%v)  env(%s) service_id(%s) hostname(%s) code(%v)",
			uri, c.Env, ServiceId, c.Host, res.Code)
		err = ec
		return
	}
	logger.Info("nmidr cancel client.Get(%v)  env(%s) service_id(%s) hostname(%s) success",
		uri, c.Env, ServiceId, c.Host)

	data = res.Data

	return
}

func (nr *Nmidr) doWatch(ctx context.Context, ServiceId string) (data registry.ReturnWatch, err error) {
	nr.mutex.RLock()
	c := nr.conf
	nr.mutex.RUnlock()

	uri := fmt.Sprintf(_watchURL, nr.pickMasterNode())
	params := nr.newParams(c)
	params.Set("service_id", ServiceId)

	res := new(struct {
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    registry.ReturnWatch `json:"data"`
	})
	if err = nr.httpClient.Post(ctx, uri, "", params, &res); err != nil {
		logger.Error("nmidr cancel client.Get(%v) env(%s) service_id(%s) hostname(%s) error(%v)",
			uri, c.Env, ServiceId, c.Host, err)
		return
	}
	if ec := ecode.Int(res.Code); !ecode.EqualError(ecode.OK, ec) {
		logger.Warn("nmidr cancel client.Get(%v)  env(%s) service_id(%s) hostname(%s) code(%v)",
			uri, c.Env, ServiceId, c.Host, res.Code)
		err = ec
		return
	}
	logger.Info("nmidr cancel client.Get(%v)  env(%s) service_id(%s) hostname(%s) success",
		uri, c.Env, ServiceId, c.Host)

	data = res.Data

	return
}

// Scheme return Nmidr's scheme
func (nr *Nmidr) Scheme() string {
	return "Nmidr"
}
