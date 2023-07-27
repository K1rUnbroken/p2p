package service

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Header struct {
	Type       int // 占1个byte，表明该数据帧的类型
	Seq        int // 占2个byte，数据片序号，用于下载文件时传输数据片，其余情况为0
	PayloadLen int // 占2个byte，表名payload部分的大小
}

type Frame struct {
	Header  *Header
	Payload []byte
}

func (f Frame) ToBytes() ([]byte, error) {
	typ := []byte(fmt.Sprintf("%01d", f.Header.Type))
	if len(typ) != 1 {
		return nil, errors.New("FrameToBytes: type length must be 1")
	}
	seq := []byte(fmt.Sprintf("%02d", f.Header.Seq))
	payloadLen := []byte(fmt.Sprintf("%02d", f.Header.PayloadLen))

	var res []byte
	res = append(res, typ...)
	res = append(res, seq...)
	res = append(res, payloadLen...)
	res = append(res, f.Payload...)

	return res, nil
}

func AddrStrToUDPAddr(addr string) *net.UDPAddr {
	port, _ := strconv.Atoi(strings.Split(addr, ":")[1])
	dstAddr := &net.UDPAddr{
		IP:   net.ParseIP(strings.Split(addr, ":")[0]),
		Port: port,
	}

	return dstAddr
}

func ParseHeader(d []byte) (*Header, error) {
	typ, _ := strconv.Atoi(string(d[0]))
	seq, _ := strconv.Atoi(string(d[1:3]))
	payloadLen, _ := strconv.Atoi(string(d[3:5]))

	return &Header{
		Type:       typ,
		Seq:        seq,
		PayloadLen: payloadLen,
	}, nil
}

func GetFrameBytes(typ, seq int, payload string) ([]byte, error) {
	payld := []byte(payload)
	// 构建frame
	f := &Frame{
		Header: &Header{
			Seq:        seq,
			Type:       typ,
			PayloadLen: len(payld),
		},
		Payload: payld,
	}

	// frame转换为[]byte
	d, err := f.ToBytes()

	return d, err
}
