package direct

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

// PoolClient manages a pool of TCP connections for concurrent multiplexing.
// Unlike Client which locks a single connection for the entire Call duration,
// PoolClient borrows/returns connections from an internal pool.
type PoolClient struct {
	Network  string
	Addr     string
	Timeout  time.Duration
	PoolSize int

	mu      sync.Mutex
	conns   chan *pooledConn
	created int
	dialErr error // sticky dial error
}

type pooledConn struct {
	conn net.Conn
	mu   sync.Mutex
}

// NewPoolClient creates a PoolClient with the given pool size.
func NewPoolClient(network, addr string, poolSize int) *PoolClient {
	if poolSize <= 0 {
		poolSize = 32
	}
	c := &PoolClient{
		Network:  network,
		Addr:     addr,
		Timeout:  1500 * time.Millisecond,
		PoolSize: poolSize,
		conns:    make(chan *pooledConn, poolSize),
	}
	return c
}

// dial creates and registers one new connection.
func (c *PoolClient) dial() (*pooledConn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.dialErr != nil {
		return nil, c.dialErr
	}
	if c.created >= c.PoolSize {
		return nil, errors.New("pool full")
	}

	conn, err := net.DialTimeout(c.Network, c.Addr, 3*time.Second)
	if err != nil {
		c.dialErr = err
		return nil, err
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}
	c.created++
	pc := &pooledConn{conn: conn}
	return pc, nil
}

// getConn gets a connection from the pool, or creates one if under limit.
// If the pool is at capacity and all connections are busy, it blocks.
func (c *PoolClient) getConn() (*pooledConn, error) {
	// First, try to create a new connection if under limit
	pc, err := c.dial()
	if err == nil {
		return pc, nil
	}
	// Pool is full or dial failed. Block until a connection is returned.
	for {
		select {
		case pc := <-c.conns:
			if pc.conn != nil {
				return pc, nil
			}
			// Dead connection, decrement count
			c.mu.Lock()
			c.created--
			c.mu.Unlock()
		default:
			// Channel empty, wait a bit
			time.Sleep(time.Microsecond)
		}
	}
}

// putConn returns a connection to the pool.
func (c *PoolClient) putConn(pc *pooledConn) {
	if pc == nil || pc.conn == nil {
		return
	}
	select {
	case c.conns <- pc:
	default:
		pc.conn.Close()
	}
}

// Call executes a request using a connection from the pool.
func (c *PoolClient) Call(service string, payload []byte) ([]byte, error) {
	pc, err := c.getConn()
	if err != nil {
		return nil, err
	}

	// Lock this specific connection for the duration of this call
	pc.mu.Lock()
	defer pc.mu.Unlock()

	conn := pc.conn
	if conn == nil {
		c.putConn(nil)
		return nil, errors.New("conn nil")
	}

	req := PackFrame(ConnTypeDirectClient, DTCall, PackCall(service, payload))

	if c.Timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(c.Timeout))
	}

	if err := writeAll(conn, req); err != nil {
		conn.Close()
		pc.conn = nil
		c.putConn(nil)
		c.mu.Lock()
		c.created--
		c.mu.Unlock()
		return nil, err
	}

	header := make([]byte, FrameHeaderSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		conn.Close()
		pc.conn = nil
		c.putConn(nil)
		c.mu.Lock()
		c.created--
		c.mu.Unlock()
		return nil, err
	}

	connType := binary.BigEndian.Uint32(header[:4])
	if connType != ConnTypeDirectServer {
		conn.Close()
		pc.conn = nil
		c.putConn(nil)
		c.mu.Lock()
		c.created--
		c.mu.Unlock()
		return nil, errors.New("invalid conn type")
	}

	dt := binary.BigEndian.Uint32(header[4:8])
	if dt != DTReply {
		conn.Close()
		pc.conn = nil
		c.putConn(nil)
		c.mu.Lock()
		c.created--
		c.mu.Unlock()
		return nil, errors.New("invalid data type")
	}

	clen := int(binary.BigEndian.Uint32(header[8:FrameHeaderSize]))
	if clen < 0 || clen > 32*1024*1024 {
		conn.Close()
		pc.conn = nil
		c.putConn(nil)
		c.mu.Lock()
		c.created--
		c.mu.Unlock()
		return nil, errors.New("invalid reply size")
	}

	content := make([]byte, clen)
	if clen > 0 {
		if _, err := io.ReadFull(conn, content); err != nil {
			conn.Close()
			pc.conn = nil
			c.putConn(nil)
			c.mu.Lock()
			c.created--
			c.mu.Unlock()
			return nil, err
		}
	}

	// Success! Return connection to pool
	c.putConn(pc)

	reply, err := UnpackReply(content)
	if err != nil {
		return nil, err
	}
	if reply.Status != 0 {
		return nil, errors.New(string(reply.Data))
	}
	return reply.Data, nil
}

// Close closes all connections in the pool.
func (c *PoolClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	close(c.conns)
	for pc := range c.conns {
		if pc.conn != nil {
			pc.conn.Close()
		}
	}
	c.created = 0
	return nil
}

// Stats returns pool stats.
func (c *PoolClient) Stats() (created int, idle int) {
	c.mu.Lock()
	created = c.created
	c.mu.Unlock()
	idle = len(c.conns)
	return
}
