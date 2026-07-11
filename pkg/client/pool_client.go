package client

import (
	"errors"
	"sync"
	"time"

	"github.com/HughNian/nmid/pkg/model"
)

// PoolClient wraps multiple Client instances sharing a connection pool.
// Each call borrows an idle Client, uses its Do(), and returns it.
// This allows multiple concurrent calls without creating new TCP connections per request.
type PoolClient struct {
	network  string
	addr     string
	timeout  time.Duration
	poolSize int

	mu      sync.Mutex
	clients chan *Client
	created int
}

// NewPoolClient creates a multi-connection client pool.
// poolSize: max concurrent in-flight requests (one per pooled Client).
func NewPoolClient(network, addr string, poolSize int) *PoolClient {
	if poolSize <= 0 {
		poolSize = 32
	}
	return &PoolClient{
		network:  network,
		addr:     addr,
		timeout:  model.DEFAULT_TIME_OUT,
		poolSize: poolSize,
		clients:  make(chan *Client, poolSize),
	}
}

// SetIoTimeOut sets read/write timeout for the pool.
func (pc *PoolClient) SetIoTimeOut(t time.Duration) *PoolClient {
	pc.timeout = t
	return pc
}

// getOrCreate returns an idle Client or creates a new one if under limit.
// Blocks if pool is at capacity and all clients are busy.
func (pc *PoolClient) getOrCreate() (*Client, error) {
	// Phase 1: try to create a new connection if under limit
	pc.mu.Lock()
	if pc.created < pc.poolSize {
		pc.created++
		pc.mu.Unlock()

		c, err := NewClient(pc.network, pc.addr).SetIoTimeOut(pc.timeout).Start()
		if err != nil || c == nil {
			pc.mu.Lock()
			pc.created--
			pc.mu.Unlock()
			// fall through to grab from channel
		} else {
			return c, nil
		}
	} else {
		pc.mu.Unlock()
	}

	// Phase 2: pool at capacity, block until a client is returned
	c := <-pc.clients
	if c != nil && c.conn != nil {
		return c, nil
	}
	// Dead client
	pc.mu.Lock()
	if pc.created > 0 {
		pc.created--
	}
	pc.mu.Unlock()
	c.Close()

	// Try again recursively
	return pc.getOrCreate()
}

// put returns a Client to the pool.
func (pc *PoolClient) put(c *Client) {
	if c == nil {
		return
	}
	// Check connection still alive
	c.Lock()
	alive := c.conn != nil
	c.Unlock()

	if !alive {
		pc.mu.Lock()
		if pc.created > 0 {
			pc.created--
		}
		pc.mu.Unlock()
		return
	}

	select {
	case pc.clients <- c:
	default:
		c.Close()
		pc.mu.Lock()
		if pc.created > 0 {
			pc.created--
		}
		pc.mu.Unlock()
	}
}

// Do executes a function call using a borrowed Client from the pool.
// This is the multi-connection multiplexing version of Client.Do().
func (pc *PoolClient) Do(funcName string, params []byte, callback RespHandler) error {
	c, err := pc.getOrCreate()
	if err != nil {
		return err
	}
	defer pc.put(c)

	return c.Do(funcName, params, callback)
}

// Close closes all pooled connections.
func (pc *PoolClient) Close() {
	close(pc.clients)
	for c := range pc.clients {
		if c != nil {
			c.Close()
		}
	}
}

// Stats returns pool statistics.
func (pc *PoolClient) Stats() (created int, idle int) {
	pc.mu.Lock()
	created = pc.created
	pc.mu.Unlock()
	idle = len(pc.clients)
	return
}

// ----------

// 原有的 RespHandler/ErrHandler 类型直接复用 client.go 中的定义

var (
	_ = errors.New // keep import
)
