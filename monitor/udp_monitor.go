package monitor

import (
	"fmt"
	"log"
	"net"

	"github.com/jamsa/hgap/packet"
)

// UDPContent 完整内容
type UDPContent struct {
	NetContent
}

// UDPMonitor UDP包监视
type UDPMonitor struct {
	NetMonitor
}

func (monitor *UDPMonitor) readPacket(buf []byte, n int) {
	//log.Printf("接收包长度:%v\n", n)
	pack := &packet.Packet{}
	err := pack.Decode(buf[:n])
	//err := gob.NewDecoder(bytes.NewReader(buf[:n])).Decode(pack)
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
