package monitor

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/jamsa/hgap/packet"
)

// TCPContent 完整内容
type TCPContent struct {
	NetContent
}

// TCPMonitor TCP包监视
type TCPMonitor struct {
	NetMonitor
}

func splitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	//FrameMagic+FrameType+Length 3个int32的长度
	if !atEOF &&
		len(data) > 4*3 &&
		binary.BigEndian.Uint32(data[:4]) == packet.FrameMagic {
		var frameType, length int32
		err = binary.Read(bytes.NewReader(data[4:8]), binary.BigEndian, &length)
		if err != nil {
			log.Errorf("读取FrameLength出错%v", length)
			return 0, nil, err
		}
		err := binary.Read(bytes.NewReader(data[8:12]), binary.BigEndian, &frameType)
		if err != nil {
			log.Errorf("读取FrameType出错%v", frameType)
			return 0, nil, err
		}
		end := 4*3 + length
		if len(data) >= int(end) {
			log.Debugf("读取帧:%d,%d,%d,%d", frameType, length, end, len(data))
			//消费end长的数据，返回从第4位开始的完整Frame数据
			return int(end), data[4:end], nil
		}
	}
	return
}

func (monitor *TCPMonitor) readFrame(conn net.Conn) error {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	//TODO 数字常量定义，buf的大小应与MTU相关
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	scanner.Split(splitFunc)
	for scanner.Scan() {
		data := scanner.Bytes()
		frame := &packet.Frame{}
		err := frame.Decode(data)
		if err != nil {
			return err
		}

		log.Debugf("收到数据帧:%v,%v", frame.FrameType, frame.Length)
		if frame.FrameType == packet.FrameTypeCLOSE {
			log.Printf("接收到关闭通知")
			//写接收标识
			//conn.Write([]byte{0})
			return nil
		}
		monitor.readPacket(frame)
	}
	return scanner.Err()
}

func (monitor *TCPMonitor) readPacket(frame *packet.Frame) {
	pack := &packet.Packet{}
	err := pack.Decode(frame.Data)
	if err != nil {
		log.Errorf("TCP数据帧解码错误: %s", err)
		return
	}
	log.Debugf("将数据解码为分组: %+v,%+v,%+v,%+v\n", pack.ID, pack.Length, pack.Begin, pack.Size)
	//log.Printf("解码数据帧，类型:%+v，长度:%+v\n", frame.FrameType, frame.Length)
	monitor.packetReceive(pack)
}

// Start 启动监视
func (monitor *TCPMonitor) Start(onReady OnReady) {
	monitor.onReady = onReady
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", monitor.host, monitor.port))
	if err != nil {
		log.Error("TCP监听失败", err)
		return
	}
	log.Println("开始TCP包监视", listener.Addr().String())

	go monitor.cleanUp()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error("接收连接出错:", err)
		} else {
			go monitor.readFrame(conn)
		}
	}
}
