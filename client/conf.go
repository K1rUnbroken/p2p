package client

import "net"

var (
	ServerAddr = &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 9999,
	}
)
