package transfer

import (
	"errors"

	"github.com/jamsa/hgap/config"
)

// ITransfer 数据传输器
type ITransfer interface {
	Send(string, []byte) //发送文件
	Remove(string)       //删除文件
}

// Transfer 数据传输器
type Transfer struct {
	//ITransfer
	textTransfer bool
}

// NewTransfer 创建数据传输对象
func NewTransfer(inBound bool, cfg *config.Config) (ITransfer, error) {
	var result ITransfer
	if inBound && cfg.InTransferType == "file" {
		fileTransfer := FileTransfer{
			Transfer: &Transfer{
				textTransfer: cfg.InTextTransfer,
			},
			path:     cfg.InDirectory,
			fileExt:  ".req",
			keepFile: cfg.KeepFiles,
		}
		result = &fileTransfer
		return result, nil
	}
	if !inBound && cfg.OutTransferType == "file" {
		fileTransfer := FileTransfer{
			Transfer: &Transfer{
				textTransfer: cfg.OutTextTransfer,
			},
			path:     cfg.OutDirectory,
			fileExt:  ".resp",
			keepFile: cfg.KeepFiles,
		}
		result = &fileTransfer
		return result, nil
	}
	if inBound && cfg.InTransferType == "udp" {
		fileTransfer := UDPTransfer{
			Transfer: &Transfer{
				textTransfer: cfg.InTextTransfer,
			},
			host: cfg.OutMonitorHost,
			port: cfg.OutMonitorPort,
		}
		result = &fileTransfer
		return result, nil
	}
	if !inBound && cfg.OutTransferType == "udp" {
		fileTransfer := UDPTransfer{
			Transfer: &Transfer{
				textTransfer: cfg.OutTextTransfer,
			},
			host: cfg.InMonitorHost,
			port: cfg.InMonitorPort,
		}
		result = &fileTransfer
		return result, nil
	}
	if inBound && cfg.InTransferType == "tcp" {
		fileTransfer := TCPTransfer{
			Transfer: &Transfer{
				textTransfer: cfg.InTextTransfer,
			},
			host: cfg.OutMonitorHost,
			port: cfg.OutMonitorPort,
		}
		result = &fileTransfer
		return result, nil
	}
	if !inBound && cfg.OutTransferType == "tcp" {
		fileTransfer := TCPTransfer{
			Transfer: &Transfer{
				textTransfer: cfg.OutTextTransfer,
			},
			host: cfg.InMonitorHost,
			port: cfg.InMonitorPort,
		}
		result = &fileTransfer
		return result, nil
	}
	return nil, errors.New("无法创建Transfer")
}
