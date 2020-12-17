package monitor

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/jamsa/hgap/packet"
)

// TODO 处理UDPContent、UDPMonitor中的重复代码，除Start、readPacket外的方法代码都相同

// TCPContent 完整内容
type TCPContent struct {
	id         string
	lock       sync.Mutex
	length     int //已接收长度
	createTime time.Time
	packets    []*packet.Packet
}

// TCPMonitor TCP包监视
type TCPMonitor struct {
	IMonitor
	*Monitor
	host     string    //监听主机
	port     int       //监听端口
	contents *sync.Map //数据
	timeout  int       //等侍文件就绪的超时时间(ms)
}

func splitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	//FrameMagic+FrameType+Length 3个int32的长度
	if !atEOF &&
		len(data) > 4*3 &&
		binary.BigEndian.Uint32(data[:4]) == packet.FrameMagic {
		var frameType, length int32
		err = binary.Read(bytes.NewReader(data[4:8]), binary.BigEndian, &length)
		if err != nil {
			log.Printf("读取FrameLength出错%v", length)
			return 0, nil, err
		}
		err := binary.Read(bytes.NewReader(data[8:12]), binary.BigEndian, &frameType)
		if err != nil {
			log.Printf("读取FrameType出错%v", frameType)
			return 0, nil, err
		}
		end := 4*3 + length
		if len(data) >= int(end) {
			log.Printf("读取帧:%d,%d,%d,%d\n", frameType, length, end, len(data))
			//消费end长的数据，返回从第4位开始的完整Frame数据
			return int(end), data[4:end], nil
		}
	}
	return
}

func (monitor *TCPMonitor) readFrame(conn net.Conn) error {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
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
		log.Printf("收到数据帧:%v,%v", frame.FrameType, frame.Length)
		if frame.FrameType == packet.FrameTypeCLOSE {
			log.Printf("接收到关闭通知")
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
		log.Printf("TCP数据帧解码错误: %s", err)
		return
	}
	log.Printf("将数据解码为分组: %+v,%+v,%+v,%+v\n", pack.ID, pack.Length, pack.Begin, pack.Size)
	//log.Printf("解码数据帧，类型:%+v，长度:%+v\n", frame.FrameType, frame.Length)
	monitor.packetReceive(pack)
}

// Start 启动监视
func (monitor *TCPMonitor) Start(onReady OnReady) {
	monitor.onReady = onReady
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", monitor.host, monitor.port))
	if err != nil {
		fmt.Println("TCP监听失败", err)
		return
	}
	log.Println("开始TCP包监视", listener.Addr().String())

	go monitor.cleanUp()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("接收连接出错:", err)
		} else {
			go monitor.readFrame(conn)
		}
	}
}

// Remove 删除数据
func (monitor *TCPMonitor) Remove(reqID string) {
	log.Println("删除接收的数据" + reqID)
	monitor.contents.Delete(reqID)
}

// Read 读取数据
func (monitor *TCPMonitor) Read(reqID string) ([]byte, error) {
	return monitor.readAll(reqID)
}

// cleanUp 清理超时数据
func (monitor *TCPMonitor) cleanUp() {
	for {
		time.Sleep(time.Duration(monitor.timeout) * time.Millisecond)
		log.Println("检查并清理超时数据...")
		var timeoutIDs []string
		monitor.contents.Range(func(k, v interface{}) bool {
			c := v.(*TCPContent)
			if time.Now().Sub(c.createTime) >
				time.Duration(monitor.timeout)*time.Millisecond {
				timeoutIDs = append(timeoutIDs, c.id)
			}
			return true
		})

		// 执行清理
		for _, v := range timeoutIDs {
			monitor.Remove(v)
		}
	}
}

// 分组包接收
func (monitor *TCPMonitor) packetReceive(pack *packet.Packet) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("处理TCP分包", pack, "出错", r)
		}
	}()

	content, ok := monitor.contents.Load(pack.ID)
	if !ok {
		content = &TCPContent{
			id:         pack.ID,
			length:     0,
			createTime: time.Now(),
		}
		monitor.contents.Store(pack.ID, content)
	}
	c := content.(*TCPContent)
	c.lock.Lock()
	c.length += pack.Size
	c.packets = append(c.packets, pack)
	c.lock.Unlock()

	//接收完毕
	if c.length >= pack.Length {
		monitor.onReady(pack.ID)
	}
}

// 读取完整内容
func (monitor *TCPMonitor) readAll(reqID string) ([]byte, error) {
	content, ok := monitor.contents.Load(reqID)
	var result bytes.Buffer
	if ok {
		c := content.(*TCPContent)
		sort.Slice(c.packets, func(i, j int) bool {
			left := c.packets[i]
			right := c.packets[j]
			return left.Begin < right.Begin
		})
		for _, v := range c.packets {
			log.Println("收集数据:", v.ID, v.Begin, "/", v.Length, v.Size)
			result.Write(v.Data)
		}
		return result.Bytes(), nil
	}
	return nil, errors.New("找不到请求数据" + reqID)
}
