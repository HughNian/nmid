package direct

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type Client struct {
	Network string
	Addr    string
	Timeout time.Duration

	mu   sync.Mutex
	conn net.Conn
}

func NewClient(network, addr string) *Client {
	return &Client{
		Network: network,
		Addr:    addr,
		Timeout: 1500 * time.Millisecond,
	}
}

func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return nil
	}
	conn, err := net.DialTimeout(c.Network, c.Addr, 3*time.Second)
	if err != nil {
		return err
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}
	c.conn = conn
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client) Call(service string, payload []byte) ([]byte, error) {
	if err := c.Connect(); err != nil {
		return nil, err
	}

	req := PackFrame(ConnTypeDirectClient, DTCall, PackCall(service, payload))

	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return nil, errors.New("conn nil")
	}
	if c.Timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(c.Timeout))
	}
	if err := writeAll(conn, req); err != nil {
		_ = c.Close()
		return nil, err
	}

	header := make([]byte, FrameHeaderSize)
	if _, err := io.ReadFull(conn, header); err != nil {
		_ = c.Close()
		return nil, err
	}
	connType := binary.BigEndian.Uint32(header[:4])
	if connType != ConnTypeDirectServer {
		_ = c.Close()
		return nil, errors.New("invalid conn type")
	}
	dt := binary.BigEndian.Uint32(header[4:8])
	if dt != DTReply {
		_ = c.Close()
		return nil, errors.New("invalid data type")
	}
	clen := int(binary.BigEndian.Uint32(header[8:FrameHeaderSize]))
	if clen < 0 || clen > 32*1024*1024 {
		_ = c.Close()
		return nil, errors.New("invalid reply size")
	}
	content := make([]byte, clen)
	if clen > 0 {
		if _, err := io.ReadFull(conn, content); err != nil {
			_ = c.Close()
			return nil, err
		}
	}
	reply, err := UnpackReply(content)
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	if reply.Status != 0 {
		return nil, errors.New(string(reply.Data))
	}
	return reply.Data, nil
}

