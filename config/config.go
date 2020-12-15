package config

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/pkg/errors"
)

// Config 配置信息
type Config struct {
	Port    int `json:"port"`    //监听端口
	Timeout int `json:"timeout"` //超时时间
	//MonitoringMode    string            `json:"monitoringMode"`   //文件监控模式
	FileScanInterval  int    `json:"fileScanInterval"`  //文件扫描间隔
	FileCheckInterval int    `json:"fileCheckInterval"` //检查文件频度
	KeepFiles         bool   `json:"keepFiles"`         //保存历史文件
	InDirectory       string `json:"inDirectory"`       //请求文件保存路径
	OutDirectory      string `json:"outDirectory"`      //响应文件保存路径
	InTextTransfer    bool   `json:"inTextTransfer"`    //InBound以文本方式传输
	OutTextTransfer   bool   `json:"outTextTransfer"`   //OutBound以文本方式传输

	InMonitorHost  string `json:"inMonitorHost"`  //InBound传输监听主机
	OutMonitorHost string `json:"outMonitorHost"` //OutBound传输监听主机
	InMonitorPort  int    `json:"inMonitorPort"`  //InBound的传输端口
	OutMonitorPort int    `json:"outMonitorPort"` //OutBound的传输端口

	InTransferType  string            `json:"inTransferType"`  //InBound传输类型
	OutTransferType string            `json:"outTransferType"` //OutBound传输类型
	URLMapping      map[string]string `json:"urlMapping"`      //URL路径映射
}

// ParseConfig 解析配置文件
func ParseConfig() (*Config, error) {
	// GlobalConfig 全局配置
	var GlobalConfig = &Config{
		Port:              9090,
		Timeout:           30000,
		FileScanInterval:  300,
		FileCheckInterval: 20,
		KeepFiles:         true,
		InDirectory:       "in/req",
		OutDirectory:      "out/resp",
		InTextTransfer:    false,
		OutTextTransfer:   false,

		InMonitorHost:  "0.0.0.0",
		InMonitorPort:  9091,
		OutMonitorHost: "0.0.0.0",
		OutMonitorPort: 9092,

		InTransferType:  "file",
		OutTransferType: "file",
		URLMapping:      map[string]string{
			// "/": "http://www.baidu.com",
		},
	}
	var cfg string
	var err error
	cfg = "config.json"
	/*if len(os.Args) > 1 {
		if cfg, err = filepath.Abs(os.Args[1]); err != nil {
			return errors.WithMessagef(err, "读取配置文件 %v 出错", os.Args[1])
		}
	}*/
	info, err := os.Stat(cfg)
	if !os.IsNotExist(err) && !info.IsDir() {
		jsonFile, err := os.Open(cfg)
		if err != nil {
			return nil, errors.WithMessagef(err, "打开配置文件 %v 出错", cfg)
		}

		defer jsonFile.Close()

		byteValue, err := ioutil.ReadAll(jsonFile)
		if err != nil {
			return nil, errors.WithMessagef(err, "读取json配置文件 %v 出错", cfg)
		}

		if err = json.Unmarshal([]byte(byteValue), &GlobalConfig); err != nil {
			return nil, errors.WithMessagef(err, "解析json配置文件 %v 出错", cfg)
		}
	}

	if err = makeDir(GlobalConfig.InDirectory); err != nil {
		return nil, err
	}
	if err = makeDir(GlobalConfig.OutDirectory); err != nil {
		return nil, err
	}
	/*
		if err = makeDir(filepath.Join(GlobalConfig.InDirectory, "tmp")); err != nil {
			return err
		}
		if err = makeDir(filepath.Join(GlobalConfig.OutDirectory, "tmp")); err != nil {
			return err
		}
	*/

	return GlobalConfig, nil
}

func makeDir(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		log.Printf("目录 %s 不存在，将创建目录", dir)
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return errors.WithMessagef(err, "创建目录出错")
		}
	}
	return nil
}
