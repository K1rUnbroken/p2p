package client

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

//---------------------------------service--------------------------------

// 向服务端请求某个文件的信息
func getFileInfo(conn *net.UDPConn, fileName string) error {
	// 构建frame
	info := fmt.Sprintf("filename:%s", fileName)
	d, err := GetFrameBytes(GetFileInfo, 0, info)
	if err != nil {
		return err
	}

	// 发送
	_, err = conn.WriteToUDP(d, ServerAddr)

	return err
}

// 下载文件
func downloadFile(cli *Client, filename string) ([]byte, error) {
	// 等待server发送信息过来
	if res := <-cli.DownloadingFiles[filename].GetInfoOK; res == 0 {
		return nil, errors.New("file " + filename + " not exists")
	}
	totalSize := cli.DownloadingFiles[filename].Filesize
	clientAddrs := cli.DownloadingFiles[filename].ClientAddrs

	// 如果有已经下载了该文件的client，则从这些client获取。否则从server获取
	if len(clientAddrs) > 0 {
		// 授予这些地址权限
		for _, addr := range clientAddrs {
			cli.AuthAddr = append(cli.AuthAddr, addr)
		}

		// 通知这些地址授予自己权限
		info := strings.Join(clientAddrs, ",")
		b, err := GetFrameBytes(NeedAuth, 0, info)
		if err != nil {
			return nil, err
		}
		_, err = cli.Lis.WriteToUDP(b, ServerAddr)
		if err != nil {
			return nil, err
		}

		// 通知clients发送文件
		avgSize := totalSize / len(clientAddrs)
		for i, addr := range clientAddrs {
			if len(clientAddrs) == 1 {
				avgSize = totalSize
			}
			start := avgSize * i
			// 生成要发送的bytes
			if i == len(clientAddrs)-1 {
				avgSize = totalSize - avgSize*i
			}
			end := start + avgSize - 1
			fmt.Println(i, ": ", start, " ", end)
			payload := fmt.Sprintf("filename:%s\nstart:%d\nend:%d", filename, start, end)
			d, err := GetFrameBytes(Download, i+1, payload)
			if err != nil {
				fmt.Printf("error: gen bytes of data piece %d fail\n", i+1)
			}

			// 发送通知
			_, err = cli.Lis.WriteToUDP(d, addrStrToUDPAddr(addr))
			if err != nil {
				fmt.Printf("error: notify conn %d fail", i)
			}

			fmt.Println("request downloading file ", filename, " to ", addr, " success ")
		}
	} else {
		// 从server获取
		b, err := GetFrameBytes(Download, 0, filename)
		if err != nil {
			return nil, errors.New(fmt.Sprintln("error: get frame bytes fail: ", err))
		}
		_, _ = cli.Lis.WriteToUDP(b, ServerAddr)
	}

	// 等待文件下载完成
	<-cli.DownloadingFiles[filename].DownloadOK

	// 将数据片排序
	var data []byte
	fileData := cli.DownloadingFiles[filename].DataPiece
	fmt.Println("fileData:", fileData)
	// 只有一个数据片且为全部数据
	if len(fileData) == 1 {
		return fileData[1], nil
	}
	// 需要排序
	for i := 1; i <= len(fileData); i++ {
		data = append(data, fileData[i]...)
	}

	return data, nil
}

//---------------------------------对外提供的功能--------------------------------

func (cli *Client) DownloadFile(filename string) error {
	// 初始化
	cli.DownloadingFiles[filename] = &fileInfo{
		ClientAddrs: []string{},
		DataPiece:   map[int][]byte{},
		DownloadOK:  make(chan int),
		GetInfoOK:   make(chan int),
	}

	// 从服务端获取文件相关信息
	err := getFileInfo(cli.Lis, filename)
	if err != nil {
		panic(err)
	}

	// 下载文件
	data, err := downloadFile(cli, filename)
	fmt.Println("data:", string(data))
	if err != nil {
		return err
	}

	// 数据存到本地
	file, _ := os.OpenFile(FileRelativePath+filename, os.O_CREATE|os.O_RDWR, 0777)
	_, _ = file.WriteString(string(data))
	_ = file.Close()
	fmt.Printf("save file %s to local success\n", filename)

	// 告诉server自己已经下载了这个文件
	b, _ := GetFrameBytes(DownloadOK, 0, filename)
	_, _ = cli.Lis.WriteToUDP(b, ServerAddr)

	// 删除记录
	delete(cli.DownloadingFiles, filename)

	return nil
}
