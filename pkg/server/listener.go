package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
)

type MakeListener func(ser *Server, address string) (ln net.Listener, err error)

var listenerMaps = make(map[string]MakeListener)

func init() {
	listenerMaps["tcp"] = tcpListener("tcp")
	listenerMaps["tcp4"] = tcpListener("tcp4")
	listenerMaps["tcp6"] = tcpListener("tcp6")
	listenerMaps["http"] = tcpListener("tcp")
	listenerMaps["ws"] = tcpListener("tcp")
	listenerMaps["wss"] = tcpListener("tcp")
}

//MakeListener make listener
func (ser *Server) MakeListener(network, address string) (ln net.Listener, err error) {
	ml := listenerMaps[network]
	if ml == nil {
		return nil, fmt.Errorf("can not make listener for %s", network)
	}

	if network == "wss" && ser.TlsConfig == nil {
		return nil, errors.New("wss must set tlsConfig")
	}

	return ml(ser, address)
}

//tcpListener tcp listener
func tcpListener(network string) MakeListener {
	return func(ser *Server, address string) (ln net.Listener, err error) {
		if ser.TlsConfig == nil {
			ln, err = net.Listen(network, address)
		} else {
			ln, err = tls.Listen(network, address, ser.TlsConfig)
		}

		return ln, err
	}
}
