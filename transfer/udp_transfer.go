package transfer

import (
	"net"

	log "github.com/sirupsen/logrus"

	"github.com/jamsa/hgap/packet"
)

// UDPTransfer 传输
type UDPTransfer struct {
	NetTransfer
}

// Send 发送文件
func (transfer *UDPTransfer) Send(reqID string, data []byte) {
	log.Printf("向%v:%v发送:%v", transfer.host, transfer.port, reqID)
	sip := net.ParseIP(transfer.host)
	srcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	dstAddr := &net.UDPAddr{IP: sip, Port: transfer.port}
	conn, err := net.DialUDP("udp", srcAddr, dstAddr)
	if err != nil {
		log.Println("连接UDP服务器失败", err)
		return
	}
	defer conn.Close()

	iter := packet.NewIterator(reqID, data, packet.MTU)
	for iter.HasNext() {
		pack := iter.Next()
		data, err := pack.Encode()
		/*
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			//T ODO 错误处理
			err := enc.Encode(pack)
		*/
		if err != nil {
			log.Println("包编码出错", err)
			continue
		}
		//len, err := conn.Write(buf.Bytes())
		len, err := conn.Write(data)
		if err != nil {
			log.Println("包发送失败", err)
			continue
		}
		//time.Sleep(time.Duration)
		log.Printf("发送分组: %+v,%+v,%+v,%+v,%v\n", pack.ID, pack.Length, pack.Begin, pack.Size, len)
	}
}
