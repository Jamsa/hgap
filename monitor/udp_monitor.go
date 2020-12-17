package monitor

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/jamsa/hgap/packet"
)

// UDPContent 完整内容
type UDPContent struct {
	id         string
	lock       sync.Mutex
	length     int //已接收长度
	createTime time.Time
	packets    []*packet.Packet
}

// UDPMonitor UDP包监视
type UDPMonitor struct {
	IMonitor
	*Monitor
	host     string    //监听主机
	port     int       //监听端口
	contents *sync.Map //数据
	timeout  int       //等侍文件就绪的超时时间(ms)
}

func (monitor *UDPMonitor) readPacket(buf []byte, n int) {
	//log.Printf("接收包长度:%v\n", n)
	pack := &packet.Packet{}
	err := gob.NewDecoder(bytes.NewReader(buf[:n])).Decode(pack)
	if err != nil {
		log.Printf("UDP数据包解码错误: %s", err)
		return
		//continue
	}
	log.Printf("接收分组: %+v,%+v,%+v,%+v,%v\n", pack.ID, pack.Length, pack.Begin, pack.Size, n)
	monitor.packetReceive(pack)
}

// Start 启动监视
func (monitor *UDPMonitor) Start(onReady OnReady) {
	monitor.onReady = onReady
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(monitor.host), Port: monitor.port})
	if err != nil {
		fmt.Println("UDP监听失败", err)
		return
	}
	log.Println("开始UDP包监视", listener.LocalAddr().String())

	go monitor.cleanUp()
	for {
		buf := make([]byte, packet.MTU*2)

		n, _, err := listener.ReadFromUDP(buf)
		if err != nil {
			log.Printf("UDP数据读取错误: %s", err)
			continue
		}
		go monitor.readPacket(buf, n)
	}
}

// Remove 删除数据
func (monitor *UDPMonitor) Remove(reqID string) {
	log.Println("删除接收的数据" + reqID)
	monitor.contents.Delete(reqID)
}

// Read 读取数据
func (monitor *UDPMonitor) Read(reqID string) ([]byte, error) {
	return monitor.readAll(reqID)
}

// cleanUp 清理超时数据
func (monitor *UDPMonitor) cleanUp() {
	for {
		time.Sleep(time.Duration(monitor.timeout) * time.Millisecond)
		log.Println("检查并清理超时数据...")
		var timeoutIDs []string
		monitor.contents.Range(func(k, v interface{}) bool {
			c := v.(*UDPContent)
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
func (monitor *UDPMonitor) packetReceive(pack *packet.Packet) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("处理UDP分包", pack, "出错", r)
		}
	}()

	content, ok := monitor.contents.Load(pack.ID)
	if !ok {
		content = &UDPContent{
			id:         pack.ID,
			length:     0,
			createTime: time.Now(),
		}
		monitor.contents.Store(pack.ID, content)
	}
	c := content.(*UDPContent)
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
func (monitor *UDPMonitor) readAll(reqID string) ([]byte, error) {
	content, ok := monitor.contents.Load(reqID)
	var result bytes.Buffer
	if ok {
		c := content.(*UDPContent)
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
