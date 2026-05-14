package direct

import (
	"net"
	"testing"
	"time"
)

func BenchmarkPackUnpackCall(b *testing.B) {
	payload := []byte(`{"name":"nmid"}`)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c := PackCall("ToUpper", payload)
		_, _ = UnpackCall(c)
	}
}

func BenchmarkDirectCallPipe(b *testing.B) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	s := NewServer("pipe", "unused")
	s.Register("ToUpper", func(payload []byte) ([]byte, error) {
		return payload, nil
	})
	go s.handleConn(serverConn)

	c := &Client{
		Network: "pipe",
		Addr:    "unused",
		Timeout: 2 * time.Second,
		conn:    clientConn,
	}

	payload := []byte(`{"name":"nmid"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Call("ToUpper", payload)
		if err != nil {
			b.Fatal(err)
		}
	}
}

