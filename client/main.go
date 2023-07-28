package main

import (
	"flag"
	"p2p2/client/client"
)

var filename = flag.String("filename", "test.txt", "switch the file you want to download")
var ip = flag.String("ip", "127.0.0.1", "set your ip")
var port = flag.Int("port", 0, "set your port")

func main() {
	flag.Parse()
	cli, err := client.NewClient(*ip, *port)
	if err != nil {
		panic(err)
		return
	}

	err = cli.DownloadFile(*filename)
	if err != nil {
		panic(err)
	}

	for {
	}
}
