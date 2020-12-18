package transfer

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jamsa/hgap/packet"
)

// TCPTransfer 传输
type TCPTransfer struct {
	NetTransfer
}

func sendFrame(conn net.Conn, frame *packet.Frame) error {
	buf, err := frame.Encode()
	if err != nil {
		log.Error("TCP帧编码出错", err)
		return err
	}
	_, err = conn.Write(buf)
	if err != nil {
		log.Error("TCP帧发送失败", err)
		return err
	}
	return nil
}

// Send 发送文件
func (transfer *TCPTransfer) Send(reqID string, data []byte) {
	log.Printf("向%v:%v发送:%v", transfer.host, transfer.port, reqID)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", transfer.host, transfer.port), time.Second*30)
	if err != nil {
		log.Error("连接TCP服务器失败", err)
		return
	}
	//log.Printf("向%s建立tcp连接", fmt.Sprintf("%s:%d", transfer.host, transfer.port))
	defer conn.Close()

	iter := packet.NewIterator(reqID, data, packet.MTU*100)
	for iter.HasNext() {
		pack := iter.Next()
		data, err := pack.Encode()
		if err != nil {
			log.Error("TCP包编码出错", err)
			continue
		}

		frame := &packet.Frame{
			FrameType: packet.FrameTypeDATA,
			Length:    int32(len(data)),
			Data:      data,
		}
		err = sendFrame(conn, frame)
		if err != nil {
			continue
		}
		//log.Printf("发送分组: %+v,%+v,%+v,%+v\n", pack.ID, pack.Length, pack.Begin, pack.Size)
		log.Debugf("发送TCP帧数据，类型:%+v,长度:%+v,数据长:%v", frame.FrameType, frame.Length, len(data))
	}

	//发送关闭通知
	frame := &packet.Frame{
		FrameType: packet.FrameTypeCLOSE,
		Length:    int32(0),
		Data:      nil,
	}
	sendFrame(conn, frame)
	if err != nil {
		return
	}
	log.Debugf("发送关闭通知:%s", reqID)

	//等侍结束位
	/*buf := make([]byte, 0, 1024)
	conn.Read(buf)*/
	log.Println("关闭传输连接", reqID)
}
