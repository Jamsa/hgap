package monitor

import (
	"errors"
	"sync"

	"github.com/jamsa/hgap/config"
)

// IMonitor 数据监听器
type IMonitor interface {
	Start(OnReady)
	Read(string) ([]byte, error)
	Remove(string)
	//SetOnReady(OnReady)
}

// OnReady 数据监听器回调
type OnReady func(string)

// Monitor 数据监听器
type Monitor struct {
	textTransfer bool //纯文本传输(base64)
	onReady      OnReady
}

// SetOnReady 设置回调
/*func (monitor *Monitor) SetOnReady(onReady OnReady) {
	monitor.onReady = onReady
}*/

// Start ss
/*func (monitor *Monitor) Start() {
	fmt.Println("###############")
}*/

// NewMonitor 创建数据监听器
func NewMonitor(inBound bool, cfg *config.Config) (IMonitor, error) {
	var result IMonitor
	if inBound && cfg.OutTransferType == "file" {
		fileMonitor := FileMonitor{
			Monitor: &Monitor{
				textTransfer: cfg.OutTextTransfer,
			},
			path:          cfg.OutDirectory,
			scanInterval:  cfg.FileScanInterval,
			timeout:       cfg.Timeout,
			checkInterval: cfg.FileCheckInterval,
			fileExt:       ".resp",
			keepFile:      cfg.KeepFiles,
		}
		result = &fileMonitor
		return result, nil
	}
	if !inBound && cfg.InTransferType == "file" {
		fileMonitor := FileMonitor{
			Monitor: &Monitor{
				textTransfer: cfg.InTextTransfer,
			},
			path:          cfg.InDirectory,
			scanInterval:  cfg.FileScanInterval,
			timeout:       cfg.Timeout,
			checkInterval: cfg.FileCheckInterval,
			fileExt:       ".req",
			keepFile:      cfg.KeepFiles,
		}
		result = &fileMonitor
		return result, nil
	}
	if inBound && cfg.OutTransferType == "udp" {
		fileMonitor := UDPMonitor{
			Monitor: &Monitor{
				textTransfer: cfg.OutTextTransfer,
			},
			host:     cfg.InMonitorHost,
			port:     cfg.InMonitorPort,
			timeout:  cfg.Timeout,
			contents: &sync.Map{},
		}
		result = &fileMonitor
		return result, nil
	}
	if !inBound && cfg.InTransferType == "udp" {
		fileMonitor := UDPMonitor{
			Monitor: &Monitor{
				textTransfer: cfg.InTextTransfer,
			},
			host:     cfg.OutMonitorHost,
			port:     cfg.OutMonitorPort,
			timeout:  cfg.Timeout,
			contents: &sync.Map{},
		}
		result = &fileMonitor
		return result, nil
	}
	return nil, errors.New("无法创建Monitor")
}
