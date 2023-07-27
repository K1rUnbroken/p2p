package main

import (
	"fmt"
	"os"
	"p2p2/client"
)

func main() {
	cli, err := client.NewClient("127.0.0.1", 8888)
	if err != nil {
		panic(err)
	}
	_ = cli.ConnectServer("udp", client.ServerAddr)

	err = cli.DownloadFile("test.txt")
	if err != nil {
		fmt.Println(err)
	}

	cli2, _ := client.NewClient("127.0.0.1", 8887)
	_ = cli2.ConnectServer("udp", client.ServerAddr)
	_ = cli2.DownloadFile("test.txt")

	cli3, _ := client.NewClient("127.0.0.1", 8886)
	_ = cli3.ConnectServer("udp", client.ServerAddr)
	_ = cli3.DownloadFile("test.txt")
}

func init() {
	_, err := os.Stat("download/")
	if err != nil {
		_ = os.Mkdir("download", 0777)
		return
	}
}
