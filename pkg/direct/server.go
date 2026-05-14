package direct

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type Handler func(payload []byte) (reply []byte, err error)

type Server struct {
	Network string
	Addr    string

	mu       sync.RWMutex
	handlers map[string]Handler
}

func NewServer(network, addr string) *Server {
	return &Server{
		Network:  network,
		Addr:     addr,
		handlers: make(map[string]Handler),
	}
}

func (s *Server) Register(name string, h Handler) {
	if name == "" || h == nil {
		return
	}
	s.mu.Lock()
	s.handlers[name] = h
	s.mu.Unlock()
}

func (s *Server) Serve(l net.Listener) error {
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(c)
	}
}

func (s *Server) ListenAndServe() error {
	ln, err := net.Listen(s.Network, s.Addr)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

func (s *Server) handleConn(c net.Conn) {
	defer c.Close()
	_ = c.SetReadDeadline(time.Time{})
	_ = c.SetWriteDeadline(time.Time{})

	header := make([]byte, FrameHeaderSize)
	for {
		if _, err := io.ReadFull(c, header); err != nil {
			return
		}
		connType := binary.BigEndian.Uint32(header[:4])
		if connType != ConnTypeDirectClient {
			return
		}
		dt := binary.BigEndian.Uint32(header[4:8])
		clen := int(binary.BigEndian.Uint32(header[8:FrameHeaderSize]))
		if clen < 0 || clen > 32*1024*1024 {
			return
		}
		content := make([]byte, clen)
		if clen > 0 {
			if _, err := io.ReadFull(c, content); err != nil {
				return
			}
		}

		switch dt {
		case DTCall:
			call, err := UnpackCall(content)
			if err != nil {
				_ = writeAll(c, PackFrame(ConnTypeDirectServer, DTReply, PackReply(1, []byte("bad request"))))
				continue
			}
			h := s.getHandler(call.Service)
			if h == nil {
				_ = writeAll(c, PackFrame(ConnTypeDirectServer, DTReply, PackReply(2, []byte("no such service"))))
				continue
			}
			reply, err := h(call.Payload)
			if err != nil {
				_ = writeAll(c, PackFrame(ConnTypeDirectServer, DTReply, PackReply(3, []byte(err.Error()))))
				continue
			}
			_ = writeAll(c, PackFrame(ConnTypeDirectServer, DTReply, PackReply(0, reply)))
		case DTHealth:
			_ = writeAll(c, PackFrame(ConnTypeDirectServer, DTReply, PackReply(0, []byte("OK"))))
		default:
			_ = writeAll(c, PackFrame(ConnTypeDirectServer, DTReply, PackReply(1, []byte("unsupported"))))
		}
	}
}

func (s *Server) getHandler(name string) Handler {
	s.mu.RLock()
	h := s.handlers[name]
	s.mu.RUnlock()
	return h
}

func writeAll(c net.Conn, b []byte) error {
	if c == nil {
		return errors.New("nil conn")
	}
	var n int
	var err error
	for i := 0; i < len(b); i += n {
		n, err = c.Write(b[i:])
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrUnexpectedEOF
		}
	}
	return nil
}

