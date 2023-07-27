package main

import (
	"log"
	"net"
	"p2p2/server/service"
)

func main() {
	server := &service.P2PServer{
		FileClients:  map[string]map[string]bool{},
		AliveClients: map[string]*service.Client{},
	}

	// 运行服务
	lis, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 9999,
	})
	if err != nil {
		log.Fatal(err)
		return
	}
	log.Printf("server start on: %s", lis.LocalAddr())

	server.Lis = lis
	server.Serve()
}
