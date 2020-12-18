package outbound

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/jamsa/hgap/config"
	"github.com/jamsa/hgap/monitor"
	"github.com/jamsa/hgap/transfer"
)

// OutBound 出站服务
type OutBound struct {
	monitor    monitor.IMonitor   //监控对象
	transfer   transfer.ITransfer //传输对象
	urlMapping map[string]string  //url映射
}

// New 构造器
func New(config *config.Config) (*OutBound, error) {
	monitor, err := monitor.NewMonitor(false, config)
	if err != nil {
		return nil, err
	}
	transfer, err := transfer.NewTransfer(false, config)
	if err != nil {
		return nil, err
	}
	result := &OutBound{
		monitor:    monitor,
		transfer:   transfer,
		urlMapping: config.URLMapping,
	}
	//monitor.SetOnReady(result.processRequest)
	return result, nil
}

// Start 启动出站服务
func (outbound *OutBound) Start() {
	outbound.monitor.Start(outbound.processRequest)
}

// 重写url
func (outbound *OutBound) rewriteURL(uri string) (string, bool) {
	for k, v := range outbound.urlMapping {
		if strings.HasPrefix(uri, k) {
			url := strings.Replace(uri, k, v, 1)
			return url, true
		}
	}
	return "", false
}

// 处理请求
func (outbound *OutBound) processRequest(reqID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("处理请求文件", reqID, "出错", r)
		}
	}()

	//读取请求
	log.Println("读取请求:" + reqID)
	content, err := outbound.monitor.Read(reqID)
	if err != nil {
		log.Error("读取响应数据", reqID, "出错", err)
		return
	}
	var buf = bufio.NewReader(strings.NewReader(string(content)))
	req, err := http.ReadRequest(buf)
	if err != nil && err != io.EOF {
		log.Error("读取请求信息出错", err)
		return
	}

	if url, ok := outbound.rewriteURL(req.RequestURI); ok {
		log.Println("URL重写:", req.RequestURI, "  -->  ", url)
		//
		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Error("读取请求Body出错", err)
			return
		}
		//转发请求
		proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))
		if err != nil {
			log.Error("构造请求对象出错", err)
			return
		}
		proxyReq.Header = make(http.Header)
		for h, val := range req.Header {
			proxyReq.Header[h] = val
			//log.Println("#####:", h, "-----", val)
		}

		httpClient := &http.Client{}
		resp, err := httpClient.Do(proxyReq)
		if err != nil {
			log.Error("执行请求时出错", err)
			return
		}

		//保存响应
		content, err := httputil.DumpResponse(resp, true)
		if err != nil {
			log.Error("Dump响应信息出错", err)
			return
		}
		outbound.transfer.Send(reqID, content)
		log.Println("写入响应数据完成:" + reqID)
		return
	}
	log.Warn("无匹配的转发路径", req.RequestURI)
}
