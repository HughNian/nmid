// nmid 并发压测 v4 - PoolClient 多路复用
package main

import (
	"flag"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	cli "github.com/HughNian/nmid/pkg/client"
	"github.com/HughNian/nmid/pkg/model"
	"github.com/vmihailenco/msgpack"
)

const SERVERHOST = "127.0.0.1"
const SERVERPORT = "6808"

var (
	concurrency = flag.Int("c", 50, "并发 goroutine 数")
	totalReqs   = flag.Int("n", 10000, "总请求数")
	poolSize    = flag.Int("pool", 256, "PoolClient 连接池大小")
	serverAddr  = SERVERHOST + ":" + SERVERPORT

	successCount int64
	failCount    int64
	busyCount    int64
	latencies    []time.Duration
	latencyMu    sync.Mutex
)

func main() {
	flag.Parse()

	fmt.Printf("\n🚀 nmid 压测 v4 (PoolClient 多路复用)\n")
	fmt.Printf("   服务地址: %s\n", serverAddr)
	fmt.Printf("   并发数:   %d\n", *concurrency)
	fmt.Printf("   总请求:   %d\n", *totalReqs)
	fmt.Printf("   连接池:   %d\n", *poolSize)
	fmt.Println("=========================================")

	startTime := time.Now()

	poolClient := cli.NewPoolClient("tcp", serverAddr, *poolSize).
		SetIoTimeOut(3 * time.Second)

	var wg sync.WaitGroup
	reqChan := make(chan int, *totalReqs)

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for range reqChan {
				reqStart := time.Now()
				done := make(chan bool, 1)

				params := map[string]interface{}{"name": "nmid-stress"}
				paramsBytes, _ := msgpack.Marshal(&params)

				respHandler := func(resp *cli.Response) {
					if resp != nil && resp.DataType == model.PDT_S_RETURN_DATA && resp.RetLen != 0 {
						var retStruct model.RetStruct
						if msgpack.Unmarshal(resp.Ret, &retStruct) == nil && retStruct.Code == 0 {
							done <- true
							return
						}
					}
					done <- false
				}

				err := poolClient.Do("ToUpper", paramsBytes, respHandler)
				latency := time.Since(reqStart)
				latencyMu.Lock()
				latencies = append(latencies, latency)
				latencyMu.Unlock()

				if err == nil {
					select {
					case ok := <-done:
						if ok {
							atomic.AddInt64(&successCount, 1)
						} else {
							atomic.AddInt64(&failCount, 1)
						}
					case <-time.After(2 * time.Second):
						atomic.AddInt64(&failCount, 1)
					}
				} else {
					atomic.AddInt64(&failCount, 1)
				}
			}
		}(i)
	}

	progress := int64(0)
	go func() {
		for {
			cur := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&failCount)
			if cur > progress && cur%1000 == 0 {
				progress = cur
				fmt.Printf("\r⏳ %d/%d", cur, *totalReqs)
			}
			if int(cur) >= *totalReqs {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()

	for i := 0; i < *totalReqs; i++ {
		reqChan <- i
	}
	close(reqChan)
	wg.Wait()

	poolClient.Close()

	elapsed := time.Since(startTime)
	done := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&failCount)

	fmt.Printf("\r📊 压测结果                                    \n")
	fmt.Println("=========================================")
	fmt.Printf("总耗时:       %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("实际完成:     %d / %d\n", done, *totalReqs)
	fmt.Printf("成功:         %d\n", atomic.LoadInt64(&successCount))
	fmt.Printf("失败:         %d\n", atomic.LoadInt64(&failCount))
	fmt.Printf("成功率:       %.2f%%\n", float64(successCount)/float64(done)*100)
	fmt.Printf("QPS:          %.0f req/s\n", float64(done)/elapsed.Seconds())

	created, idle := poolClient.Stats()
	fmt.Printf("连接池:       created=%d idle=%d\n", created, idle)

	latencyMu.Lock()
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		var totalLatency time.Duration
		for _, l := range latencies {
			totalLatency += l
		}
		avg := totalLatency / time.Duration(len(latencies))
		p50 := latencies[len(latencies)*50/100]
		p95 := latencies[len(latencies)*95/100]
		p99 := latencies[len(latencies)*99/100]

		fmt.Println("---")
		fmt.Printf("平均: %v  |  P50: %v  |  P95: %v  |  P99: %v\n",
			avg.Round(time.Microsecond),
			p50.Round(time.Microsecond),
			p95.Round(time.Microsecond),
			p99.Round(time.Microsecond))
		fmt.Printf("最小: %v  |  最大: %v\n",
			latencies[0].Round(time.Microsecond),
			latencies[len(latencies)-1].Round(time.Microsecond))
	}
	latencyMu.Unlock()
	fmt.Println("=========================================")
}
