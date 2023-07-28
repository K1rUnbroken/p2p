package service

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

const (
	fileRelativePath = "server/"
	MaxSize          = 10
	PingPeriod       = 10 * time.Second
)

type Client struct {
	Addr     string
	NextPing time.Time
}

type P2PServer struct {
	Lis          *net.UDPConn
	AliveClients map[string]*Client
	FileClients  map[string]map[string]bool
}

func (s *P2PServer) Serve() {
	conn := s.Lis
	frame := make([]byte, 1024)

	// 心跳检测
	go ping(s)

	for {
		_, remoteAddr, err := conn.ReadFromUDP(frame)
		if err != nil {
			fmt.Println("error: read from remote udp fail", err)
			continue
		}

		// 解析header
		header, err := ParseHeader(frame[:HeaderLen])
		if err != nil {
			fmt.Println(err)
			continue
		}
		// 解析payload
		payload := frame[HeaderLen : HeaderLen+header.PayloadLen]

		switch header.Type {
		// 收到客户端连接
		case ConnectSvr:
			s.AliveClients[remoteAddr.String()] = &Client{
				Addr:     remoteAddr.String(),
				NextPing: time.Now().Add(PingPeriod),
			}
			fmt.Printf("receive from %s: %s", remoteAddr.String(), string(payload))

		// 客户端请求获取某个文件的信息
		case GetFileInfo:
			// 得到文件名
			filename := strings.Split(string(payload), ":")[1]

			// 获取文件信息
			var info string
			file, err := os.Open(fileRelativePath + filename)
			if err != nil {
				info = fmt.Sprintf("filename:%s\nfile not exists", filename)
			} else {
				fileInfo, _ := file.Stat()
				fileSize := fileInfo.Size()
				// 得到所有下载了该文件的client地址
				var addrs []string
				for addr, b := range s.FileClients[filename] {
					// 该客户端已经失活了
					if _, ok := s.AliveClients[addr]; !ok {
						delete(s.FileClients[filename], addr)
						fmt.Println("FileClients", s.FileClients["test.txt"])
						continue
					}
					// 客户端本人
					if addr == remoteAddr.String() {
						continue
					}
					if b {
						addrs = append(addrs, addr)
					}
				}
				str := strings.Join(addrs, ",")
				info = fmt.Sprintf("filename:%s\nfilesize:%d\nclients:%s", filename, fileSize, str)
			}

			// 生成[]byte
			d, err := GetFrameBytes(GetFileInfo, 0, info)
			if err != nil {
				fmt.Println(err)
				return
			}

			// 发送
			_, _ = conn.WriteToUDP(d, remoteAddr)

			_ = file.Close()

		// 客户端请求服务端，通知指定的client连接它
		case NeedAuth:
			// 解析出clients
			addrs := strings.Split(string(payload), ",")

			// 通知这些clients
			for _, addr := range addrs {
				b, err := GetFrameBytes(NeedAuth, 0, remoteAddr.String())
				dstAddr := AddrStrToUDPAddr(addr)
				if err == nil {
					_, _ = conn.WriteToUDP(b, dstAddr)
				}
			}

			fmt.Println(remoteAddr.String(), ":", addrs)

		// 客户端直接在服务端下载整个文件
		case Download:
			// 解析出文件名
			filename := string(payload)

			// 从本地读文件数据
			file, err := os.Open(fileRelativePath + filename)
			if err != nil {
				fmt.Printf("error: open file on server %s fail: %s", filename, err.Error())
				return
			}
			fileInfo, _ := file.Stat()
			size := int(fileInfo.Size())

			// 流量控制
			seq := 1
			for size > 0 {
				var fileData []byte
				if size < MaxSize {
					fileData = make([]byte, size)
				} else {
					fileData = make([]byte, MaxSize)
				}

				n, err := file.Read(fileData)
				size = size - n

				// 发送数据过去
				d, err := GetFrameBytes(DataPiece, seq, "filename:"+filename+"\n"+string(fileData))
				if err == nil {
					_, _ = conn.WriteToUDP(d, remoteAddr)
					seq++
				}
			}

			_ = file.Close()

		// 记录已经拥有某个资源的客户端
		case DownloadOK:
			// 解析出文件名
			filename := string(payload)

			fmt.Println(remoteAddr.String(), "download ok ", filename)

			// 记录
			if _, ok := s.FileClients[filename]; !ok {
				s.FileClients[filename] = make(map[string]bool, 0)
			}
			s.FileClients[filename][remoteAddr.String()] = true

		case Message:
			// 解析出消息体
			info := string(payload)

			// 心跳
			if info == "ping" {
				fmt.Println("receive ping from ", remoteAddr.String())
				s.AliveClients[remoteAddr.String()].NextPing = time.Now().Add(PingPeriod)
			}
		}
	}
}

func ping(s *P2PServer) {
	t := time.NewTicker(time.Second * 10)

	for {
		select {
		case <-t.C:
			for addr, cli := range s.AliveClients {
				if time.Now().After(cli.NextPing) {
					delete(s.AliveClients, addr)
				}
			}

			fmt.Println("AliveClients", s.AliveClients)

		}
	}
}
