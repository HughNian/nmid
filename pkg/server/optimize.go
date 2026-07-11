// nmid server 并发
//
// 问题分析:
// 1. doJob() 同步阻塞: client goroutine 写完 job 到 worker 后才能读下一个请求
// 2. PushJobToChannel 满了直接丢: 256 buffer, 高并发丢 job
// 3. processResults 单 goroutine: 所有 worker 结果排队处理
// 4. 一个 client 连接 = 一个 goroutine 串行读
//
// 优化方案:
// 1. [WorkerPool] 异步任务调度池，解耦 client 和 worker
// 2. [ResultRouter] 多 goroutine 结果路由
// 3. [JobChannel] 改为阻塞写入 + 背压
// 4. [CheckinClient] 分离 client 读和写

package server

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/HughNian/nmid/pkg/model"
)

// ============================================================
// 1. 异步任务调度池 - 解耦 client IO 和 worker 调度
// ============================================================

type DispatchPool struct {
	workers   int
	jobCh     chan *dispatchJob
	wg        sync.WaitGroup
	ctxCancel chan struct{}
}

type dispatchJob struct {
	client  *SClient
	doJobFn func()
}

func NewDispatchPool(workerCount int) *DispatchPool {
	if workerCount <= 0 {
		workerCount = 512
	}
	dp := &DispatchPool{
		workers:   workerCount,
		jobCh:     make(chan *dispatchJob, workerCount*2),
		ctxCancel: make(chan struct{}),
	}

	for i := 0; i < workerCount; i++ {
		dp.wg.Add(1)
		go dp.worker()
	}

	return dp
}

func (dp *DispatchPool) worker() {
	defer dp.wg.Done()
	for {
		select {
		case dj := <-dp.jobCh:
			dj.doJobFn()
		case <-dp.ctxCancel:
			return
		}
	}
}

func (dp *DispatchPool) Submit(fn func()) bool {
	select {
	case dp.jobCh <- &dispatchJob{doJobFn: fn}:
		return true
	default:
		return false
	}
}

func (dp *DispatchPool) Close() {
	close(dp.ctxCancel)
	dp.wg.Wait()
}

// ============================================================
// 2. 结果路由器 - 多 goroutine 并行处理 worker 返回
// ============================================================

type ResultRouter struct {
	workers   int
	resultCh  chan *dispatchResult
	wg        sync.WaitGroup
	ctxCancel chan struct{}
}

type dispatchResult struct {
	sw      *SWorker
	jobFunc string
	jobId   string
	req     *Request
}

func NewResultRouter(workerCount int) *ResultRouter {
	if workerCount <= 0 {
		workerCount = 256
	}
	rr := &ResultRouter{
		workers:   workerCount,
		resultCh:  make(chan *dispatchResult, workerCount*4),
		ctxCancel: make(chan struct{}),
	}

	for i := 0; i < workerCount; i++ {
		rr.wg.Add(1)
		go rr.worker()
	}

	return rr
}

func (rr *ResultRouter) worker() {
	defer rr.wg.Done()
	for {
		select {
		case dr := <-rr.resultCh:
			dr.sw.returnData(&ResultJob{
				FuncName: dr.jobFunc,
				JobId:    dr.jobId,
				Request:  dr.req,
			})
		case <-rr.ctxCancel:
			return
		}
	}
}

func (rr *ResultRouter) Submit(sw *SWorker, funcName, jobId string, req *Request) bool {
	select {
	case rr.resultCh <- &dispatchResult{sw: sw, jobFunc: funcName, jobId: jobId, req: req}:
		return true
	default:
		return false
	}
}

func (rr *ResultRouter) Close() {
	close(rr.ctxCancel)
	rr.wg.Wait()
}

// ============================================================
// 3. 改进的 SClient.doJob - 异步调度
// ============================================================

var (
	globalDispatchPool *DispatchPool
	globalDispatchOnce sync.Once
)

func GetDispatchPool() *DispatchPool {
	globalDispatchOnce.Do(func() {
		globalDispatchPool = NewDispatchPool(512)
	})
	return globalDispatchPool
}

// doJobAsync 异步版本的 doJob，不阻塞 client 的 DoIO goroutine
func (c *SClient) doJobAsync() {
	c.Req.ReqDecodePack()

	if c.Req.HandleLen == 0 || c.Req.Handle == `` {
		c.Res.DataType = model.PDT_ERROR
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)
		return
	}
	if c.Req.ParamsLen == 0 || len(c.Req.Params) == 0 {
		c.Res.DataType = model.PDT_ERROR
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)
		return
	}

	// 异步查找 worker 并调度（拷贝参数避免被下一帧覆盖）
	handle := c.Req.Handle
	params := make([]byte, len(c.Req.Params))
	copy(params, c.Req.Params)
	paramsType := c.Req.ParamsType
	paramsHandleType := c.Req.ParamsHandleType
	clientId := c.ClientId
	connect := c.Connect
	ser := c.Connect.Ser

	if !GetDispatchPool().Submit(func() {
		worker := ser.Funcs.GetBestWorker(handle)
		if worker == nil {
			res := NewRes()
			res.DataType = model.PDT_CANT_DO
			resPack := res.ResEncodePack()
			connect.Write(resPack)
			return
		}

		job := NewJobData(handle, string(params))
		job.Lock()
		job.WorkerId = worker.WorkerId
		job.Client = connect
		job.ClientId = clientId
		job.FuncName = handle
		job.Params = params
		job.ParamsType = paramsType
		job.ParamsHandleType = paramsHandleType
		job.Unlock()

		// 带背压的 job 推送
		if !worker.PushJobToChannelWithBackpressure(job, 100*time.Millisecond) {
			res := NewRes()
			res.DataType = model.PDT_CANT_DO
			resPack := res.ResEncodePack()
			connect.Write(resPack)
			return
		}

		worker.doWork(job)

		requestCount.Inc(worker.WorkerName, handle)
	}) {
		// 调度池满了，返回限流
		c.Res.DataType = model.PDT_RATELIMIT
		resPack := c.Res.ResEncodePack()
		c.Connect.Write(resPack)
	}
}

// ============================================================
// 4. 改进的 RunClient - 使用异步调度
// ============================================================

func (c *SClient) RunClientOptimized() {
	dataType := c.Req.GetReqDataType()
	switch dataType {
	case model.PDT_C_DO_JOB:
		c.doJobAsync()
	}
}

// ============================================================
// 5. Worker PushJobToChannel 带背压
// ============================================================

func (w *SWorker) PushJobToChannelWithBackpressure(job *JobData, timeout time.Duration) bool {
	if job == nil || job.FuncName == "" {
		return false
	}

	ch := w.GetJobChannel(job.FuncName)
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case ch <- job:
		return true
	case <-timer.C:
		return false
	}
}

// ============================================================
// 6. ResultRouter 集成到 SWorker
// ============================================================

var (
	globalResultRouter *ResultRouter
	globalResultOnce   sync.Once
)

func GetResultRouter() *ResultRouter {
	globalResultOnce.Do(func() {
		globalResultRouter = NewResultRouter(256)
	})
	return globalResultRouter
}

// RunWorkerOptimized 优化版 worker 处理，用 ResultRouter 异步处理结果
func (w *SWorker) RunWorkerOptimized(data []byte) {
	req := NewReq()
	req.DataType = w.Connect.DataType
	req.DataLen = w.Connect.DataLen
	req.Data = data
	req.ReqDecodePack()

	switch req.DataType {
	case model.PDT_W_ADD_FUNC:
		w.addFunction(req)
	case model.PDT_W_DEL_FUNC:
		w.delFunction(req)
	case model.PDT_WAKEUP:
		w.workerWakeup()
	case model.PDT_W_RETURN_DATA:
		// 异步路由结果，不阻塞 worker IO goroutine
		if !GetResultRouter().Submit(w, req.Handle, req.JobId, req) {
			// 路由满了，直接同步处理
			w.returnData(&ResultJob{
				FuncName: req.Handle,
				JobId:    req.JobId,
				Request:  req,
			})
		}
	case model.PDT_W_HEARTBEAT_PING:
		w.heartBeatPong()
	case model.PDT_W_SET_NAME:
		w.setWorkerName(req)
	}
}

// ============================================================
// 7. Client 读写分离 - 写操作通过 writeLoop 已有，这里加并发读支持
// ============================================================

// ReadRequest 非阻塞读请求，供并发 client 使用
// 注意：这需要改协议加 request-id 来关联请求-响应
type MultiplexClient struct {
	sync.Mutex
	connect     *Connect
	pendingReqs sync.Map // requestId → response channel
	reqId       int64
}

func NewMultiplexClient(conn *Connect) *MultiplexClient {
	return &MultiplexClient{
		connect: conn,
	}
}

func (mc *MultiplexClient) nextReqId() int64 {
	return atomic.AddInt64(&mc.reqId, 1)
}

// QueueRequest 排队一个请求，返回响应 channel
func (mc *MultiplexClient) QueueRequest(reqId int64, respCh chan []byte) {
	mc.pendingReqs.Store(reqId, respCh)
}

// HandleResponse 处理响应帧，路由到对应请求
func (mc *MultiplexClient) HandleResponse(reqId int64, data []byte) {
	if ch, ok := mc.pendingReqs.LoadAndDelete(reqId); ok {
		select {
		case ch.(chan []byte) <- data:
		default:
		}
	}
}
