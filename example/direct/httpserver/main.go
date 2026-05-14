package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/HughNian/nmid/pkg/direct"
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

var pool chan *direct.Client

func poolSize() int {
	if v := os.Getenv("DIRECT_POOL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 200
}

func serverAddr() string {
	if v := os.Getenv("DIRECT_ADDR"); v != "" {
		return v
	}
	return "127.0.0.1:7900"
}

func initPool() {
	addr := serverAddr()
	size := poolSize()
	pool = make(chan *direct.Client, size)
	for i := 0; i < size; i++ {
		c := direct.NewClient("tcp", addr)
		c.Timeout = 1500 * time.Millisecond
		pool <- c
	}
}

func getClient(timeout time.Duration) *direct.Client {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case c := <-pool:
		return c
	case <-timer.C:
		return nil
	}
}

func putClient(c *direct.Client) {
	if c == nil {
		return
	}
	select {
	case pool <- c:
	default:
		_ = c.Close()
	}
}

func Test(ctx *fasthttp.RequestCtx) {
	c := getClient(500 * time.Millisecond)
	if c == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		fmt.Fprint(ctx, "ER")
		return
	}
	defer putClient(c)

	resp, err := c.Call("ToUpper", []byte(`{"name":"nmid"}`))
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusRequestTimeout)
		fmt.Fprint(ctx, "ER")
		_ = c.Close()
		return
	}
	fmt.Println(string(resp))
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.Write(resp)
}

func main() {
	initPool()
	router := fasthttprouter.New()
	router.GET("/test", Test)
	_ = fasthttp.ListenAndServe(":5981", router.Handler)
}
