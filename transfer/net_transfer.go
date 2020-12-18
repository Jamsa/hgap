package transfer

import (
	log "github.com/sirupsen/logrus"
)

// NetTransfer 传输
type NetTransfer struct {
	ITransfer
	*Transfer
	host string //服务器主机
	port int    //服务器端口
}

// Remove 删除数据
func (transfer *NetTransfer) Remove(reqID string) {
	log.Println("删除传输的数据(NOP)" + reqID)
}
