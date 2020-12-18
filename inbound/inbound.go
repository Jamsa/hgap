package inbound

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/jamsa/hgap/config"
	"github.com/jamsa/hgap/monitor"
	"github.com/jamsa/hgap/transfer"
	uuid "github.com/satori/go.uuid"
)

// InBound 入站服务
type InBound struct {
	port     int                //监听端口
	monitor  monitor.IMonitor   //监控对象
	transfer transfer.ITransfer //传输对象
	requests *sync.Map          //请求map
	timeout  int                //超时时间
}

type finishChan chan interface{}

// New 构造器
func New(config *config.Config) (*InBound, error) {
	monitor, err := monitor.NewMonitor(true, config)
	if err != nil {
		return nil, err
	}
	transfer, err := transfer.NewTransfer(true, config)
	if err != nil {
		return nil, err
	}
	result := &InBound{
		port:     config.Port,
		monitor:  monitor,
		transfer: transfer,
		requests: &sync.Map{},
		timeout:  config.Timeout,
	}
	//monitor.SetOnReady(result.notify)
	return result, nil
}

// Start 启动入站服务
func (inbound *InBound) Start() {
	go inbound.monitor.Start(inbound.notify)

	//启动监听服务
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", inbound.port),
		ReadTimeout:  time.Duration(inbound.timeout) * time.Millisecond,
		WriteTimeout: time.Duration(inbound.timeout) * time.Millisecond,
	}
	http.HandleFunc("/", inbound.index)
	log.Println("开始监听", inbound.port, "...")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("监听出错: ", err)
	}
}

// notify 响应通知
func (inbound *InBound) notify(reqID string) {
	ch, ok := inbound.requests.Load(reqID)
	if ok {
		log.Println("发送响应文件通知:" + reqID)
		ch.(finishChan) <- struct{}{}
	} else {
		log.Println("文件响应通道不存在:" + reqID)
	}
}

// cleanUp 清理
func (inbound *InBound) cleanUp(reqID string, finish finishChan, timeout *time.Ticker) {
	timeout.Stop()
	//delete(reqs, reqID)
	inbound.requests.Delete(reqID)
	inbound.transfer.Remove(reqID)
	inbound.monitor.Remove(reqID)
	close(finish)
}

// writeResp 发送响应
func (inbound *InBound) writeResp(reqID string, respWriter http.ResponseWriter, request *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("输出响应出错", r)
		}
	}()
	//读取响应
	log.Println("读取响应:" + reqID)
	content, err := inbound.monitor.Read(reqID)
	if err != nil {
		log.Println("读取响应数据", reqID, "出错", err)
		return
	}
	var buf = bufio.NewReader(strings.NewReader(string(content)))
	resp, err := http.ReadResponse(buf, request)
	if err != nil {
		log.Println("读取Http响应信息出错", err)
		return
	}
	defer resp.Body.Close()

	for h, val := range resp.Header {
		respWriter.Header().Set(h, val[0])
	}
	b := new(bytes.Buffer)
	io.Copy(b, resp.Body)
	respWriter.Write(b.Bytes())
}

func (inbound *InBound) index(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("请求处理出错", r)
		}
	}()
	content, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Error("保存请求信息出错", err)
		return
	}
	uid, err := uuid.NewV4()
	if err != nil {
		log.Error("生成请求uuid出错", err)
		return
	}
	reqID := uid.String()
	log.Println("发送请求:" + reqID)
	inbound.transfer.Send(reqID, content)
	log.Println("请求发送完成:" + reqID)

	finish := make(finishChan)
	inbound.requests.Store(reqID, finish)
	//超时
	timeout := time.NewTicker(time.Duration(inbound.timeout) * time.Millisecond)
	defer inbound.cleanUp(reqID, finish, timeout)

	select {
	case <-finish:
		log.Println("获取响应:" + reqID)
		inbound.writeResp(reqID, w, r)
	case <-timeout.C:
		log.Warn("请求处理超时:" + reqID)
		//返回时将自动cleanUp
	}
}
