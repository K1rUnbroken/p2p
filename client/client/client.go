package client

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type fileInfo struct {
	Filesize    int            // 该文件总数据量
	NowSize     int            // 已经下载好的数据量
	ClientAddrs []string       // 拥有该文件资源的其他客户端地址
	DataPiece   map[int][]byte // 已经下载好的数据片
	DownloadOK  chan int       // 是否已经下载完毕
	GetInfoOK   chan int       // 是否已经从服务器拿到信息
}

type Client struct {
	Lis              *net.UDPConn
	AuthAddr         []string             // 有权限访问本机的remote
	DownloadingFiles map[string]*fileInfo // 正在下载中的文件
	MyAddr           *net.UDPAddr
}

//---------------------------------service--------------------------------

func addrStrToUDPAddr(addr string) *net.UDPAddr {
	port, _ := strconv.Atoi(strings.Split(addr, ":")[1])
	dstAddr := &net.UDPAddr{
		IP:   net.ParseIP(strings.Split(addr, ":")[0]),
		Port: port,
	}

	return dstAddr
}

func (cli *Client) ping() {
	for {
		b, err := GetFrameBytes(Message, 0, "ping")
		if err != nil {
			fmt.Println("send ping error: ", err)
		}
		_, err = cli.Lis.WriteToUDP(b, ServerAddr)
		if err != nil {
			fmt.Println("send ping error: ", err)
		}
		time.Sleep(time.Second * 5)
	}
}

func (cli *Client) read() {
	conn := cli.Lis
	frame := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(frame)
		if err != nil {
			fmt.Println("error: read from remote udp fail", err)
			continue
		}

		// 鉴权
		ok := false
		if remoteAddr.String() == "127.0.0.1:9999" {
			ok = true
		}
		for _, addr := range cli.AuthAddr {
			if addr == remoteAddr.String() {
				ok = true
				break
			}
		}
		if !ok {
			info := "authorization fail to connect " + cli.MyAddr.String()
			b, _ := GetFrameBytes(Message, 0, info)
			_, _ = conn.Write(b)
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

		data := string(payload)
		parts := strings.Split(data, "\n")
		switch header.Type {
		// 某个其他客户端请求下载文件
		case Download:
			// 解析出filename、start、end、seq
			seq := header.Seq
			filename := strings.Split(parts[0], ":")[1]
			start, _ := strconv.Atoi(strings.Split(parts[1], ":")[1])
			end, _ := strconv.Atoi(strings.Split(parts[2], ":")[1])

			// 取出数据
			file, err := os.Open(FileRelativePath + filename)
			if err != nil {
				fmt.Printf("error: open file %s fail: %s", filename, err.Error())
				continue
			}
			fileData := make([]byte, end-start+1)
			n, err = file.ReadAt(fileData, int64(start))
			if n != end-start+1 {
				fmt.Printf("error: read file %s uncompletable", filename)
				continue
			}
			_ = file.Close()

			// 发送数据过去
			d, err := GetFrameBytes(DataPiece, seq, "filename:"+filename+"\n"+string(fileData))
			_, _ = conn.WriteToUDP(d, remoteAddr)

			fmt.Println("receive downloading request from ", remoteAddr.String())

		// server通知自己授予某个client权限
		case NeedAuth:
			// 权限
			addr := string(payload)
			cli.AuthAddr = append(cli.AuthAddr, addr)

		// 收到文件数据片
		case DataPiece:
			// 解析出filename
			part := strings.SplitN(string(payload), "\n", 2)
			filenamePart := part[0]
			dataPart := part[1]
			filename := strings.Split(filenamePart, ":")[1]
			payload = []byte(dataPart)
			payloadLen := len(payload)

			cli.DownloadingFiles[filename].DataPiece[header.Seq] = append(cli.DownloadingFiles[filename].DataPiece[header.Seq], payload...)

			// 记录进度
			cli.DownloadingFiles[filename].NowSize += payloadLen

			fmt.Println("receive data piece from ", remoteAddr.String(), " : ", string(payload))

			// 进度
			progress := float64(cli.DownloadingFiles[filename].NowSize) / float64(cli.DownloadingFiles[filename].Filesize) * 100
			fmt.Printf("file %s downloading progress: %d%%\n", filename, int(progress))
			if progress == 100 {
				cli.DownloadingFiles[filename].DownloadOK <- 1
				fmt.Printf("file %s download success\n", filename)
				break
			}

		// 收到文件信息
		case GetFileInfo:
			// 解析出filename、filesize、clientAddrs
			filename := strings.Split(parts[0], ":")[1]

			// 文件不存在
			if len(parts) == 2 && parts[1] == "file not exists" {
				cli.DownloadingFiles[filename].GetInfoOK <- 0
				continue
			}

			filesize := strings.Split(parts[1], ":")[1]
			size, _ := strconv.Atoi(filesize)
			arr := make([]string, 0)
			clientAddrs := strings.SplitN(parts[2], ":", 2)[1]
			if clientAddrs != "" {
				arr = strings.Split(clientAddrs, ",")
			}

			cli.DownloadingFiles[filename].Filesize = size
			cli.DownloadingFiles[filename].ClientAddrs = arr
			cli.DownloadingFiles[filename].GetInfoOK <- 1

		case Message:
			fmt.Println("receive from " + remoteAddr.String() + ": " + string(payload))
		}

		time.Sleep(time.Second)
	}
}

//---------------------------------对外提供的功能--------------------------------

func NewClient(ip string, port int) (*Client, error) {
	myAddr := &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}

	conn, err := net.ListenUDP("udp", myAddr)
	if err != nil {
		return nil, err
	}
	cli := &Client{
		AuthAddr:         []string{},
		DownloadingFiles: map[string]*fileInfo{},
		MyAddr:           myAddr,
		Lis:              conn,
	}

	// 向server发送hello
	info := fmt.Sprintf("this is %s\n", cli.MyAddr.String())
	d, err := GetFrameBytes(ConnectSvr, 0, info)
	if err != nil {
		return nil, err
	}
	_, err = cli.Lis.WriteToUDP(d, ServerAddr)

	// 开启goroutine接受server信息
	go cli.read()

	// 定时向服务器发送心跳
	go cli.ping()

	return cli, nil
}
