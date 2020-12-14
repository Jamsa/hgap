package inbound

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jamsa/hgap/config"
	fsmon "github.com/jamsa/hgap/fsmon"
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
func New(config config.Config) (*InBound, error) {
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
			log.Println("请求处理出错", r)
		}
	}()
	content, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Println("保存请求信息出错", err)
		return
	}
	uid, err := uuid.NewV4()
	if err != nil {
		log.Println("生成请求uuid出错", err)
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
		log.Println("请求处理超时:" + reqID)
		//返回时将自动cleanUp
	}
}

// ===============================================

var reqs sync.Map

func fileChangeHandle(fileName string) {
	//time.Sleep(1 * time.Second)
	defer func() {
		if r := recover(); r != nil {
			log.Println("处理请求文件", fileName, "出错", r)
		}
	}()
	err := fsmon.WaitForFile(fileName)
	if err != nil {
		log.Println("等侍文件就绪时出错", err)
		return
	}
	_, file := filepath.Split(fileName)
	chName := strings.TrimSuffix(file, filepath.Ext(file))
	ch, ok := reqs.Load(chName)

	if ok {
		log.Println("发送响应文件通知:" + chName)
		ch.(chan interface{}) <- struct{}{}
	} else {
		log.Println("文件响应通道不存在:" + chName)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("请求处理出错", r)
		}
	}()
	content, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Println("保存请求信息出错", err)
		return
	}
	uid, err := uuid.NewV4()
	if err != nil {
		log.Println("生成请求uuid出错", err)
		return
	}
	reqID := uid.String()

	log.Println("保存请求:" + reqID)
	//log.Println(string(content))

	eof := "EOF" + reqID
	if config.GlobalConfig.InTextTransfer {
		content = []byte(base64.StdEncoding.EncodeToString(content))
	}
	content = append(content, []byte(eof)...)

	err = ioutil.WriteFile(filepath.Join(config.GlobalConfig.InDirectory, reqID+".req"), content, 0644)
	if err != nil {
		log.Println("写入请求文件出错", err)
		return
	}

	/*
		noti, err := os.Create(filepath.Join(config.GlobalConfig.InDirectory, reqID+".noti"))
		if err != nil {
			log.Println("写入请求文件出错", err)
			return
		}
		noti.Close()
		err = os.Rename(filepath.Join(config.GlobalConfig.InDirectory, reqID+".tmp"), filepath.Join(config.GlobalConfig.InDirectory, reqID+".req"))
		if err != nil {
			log.Println("重命名请求文件出错", err)
			return
		}
	*/

	log.Println("写入请求文件完成:" + reqID)

	finish := make(chan interface{})
	reqs.Store(reqID, finish)
	//超时
	timeout := time.NewTicker(time.Duration(config.GlobalConfig.Timeout) * time.Millisecond)

	cleanUp := func() {
		timeout.Stop()
		//delete(reqs, reqID)
		reqs.Delete(reqID)
		if !config.GlobalConfig.KeepFiles {
			os.Remove(filepath.Join(config.GlobalConfig.InDirectory, reqID+".req"))
			os.Remove(filepath.Join(config.GlobalConfig.OutDirectory, reqID+".resp"))
		}
		close(finish)
	}
	defer cleanUp()

	writeResp := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("输出响应出错", r)
			}
		}()
		//读取响应
		log.Println("读取响应:" + reqID)
		var buf *bufio.Reader
		if config.GlobalConfig.OutTextTransfer {
			content, err := ioutil.ReadFile(filepath.Join(config.GlobalConfig.OutDirectory, reqID+".resp"))
			if err != nil {
				log.Println("读取响应文件", reqID, "出错", err)
				return
			}

			content, err = base64.StdEncoding.DecodeString(string(content))
			if err != nil {
				log.Println("解码文件", reqID, "出错", err)
				return
			}

			buf = bufio.NewReader(strings.NewReader(string(content)))
		} else {
			f, err := os.Open(filepath.Join(config.GlobalConfig.OutDirectory, reqID+".resp"))
			if err != nil {
				log.Println("打开响应文件出错", err)
				return
			}
			defer f.Close()
			buf = bufio.NewReader(f)
		}

		resp, err := http.ReadResponse(buf, r)
		if err != nil {
			log.Println("读取响应信息出错", err)
			return
		}
		defer resp.Body.Close()

		//rcontent, err := httputil.DumpResponse(resp, true)
		//log.Println(rcontent)

		/*resp.Body.Close()
		b := new(bytes.Buffer)
		io.Copy(b, resp.Body)
		resp.Body.Close()*/
		for h, val := range resp.Header {
			w.Header().Set(h, val[0])
		}
		b := new(bytes.Buffer)
		io.Copy(b, resp.Body)
		//resp.Body.Close()
		//输出从响应备份文件中恢复的响应内容
		w.Write(b.Bytes())
	}

	select {
	case <-finish:
		log.Println("获取响应:" + reqID)
		writeResp()
	case <-timeout.C:
		log.Println("请求处理超时:" + reqID)

	}
	//cleanUp()
}

// Start InBound服务入口
func Start() {
	//go fsmon.StartWatcher(config.GlobalConfig.OutDirectory, fileChangeHandle)
	log.Println("开始监视响应文件目录")
	go fsmon.StartScan(config.GlobalConfig.OutDirectory, fileChangeHandle)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.GlobalConfig.Port),
		ReadTimeout:  time.Duration(config.GlobalConfig.Timeout) * time.Millisecond,
		WriteTimeout: time.Duration(config.GlobalConfig.Timeout) * time.Millisecond,
	}
	http.HandleFunc("/", index)
	//http.Server.ReadTimeout = 30 * time.Second

	log.Println("开始监听", config.GlobalConfig.Port, "...")
	//err = http.ListenAndServe(, nil)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("监听出错: ", err)
	}
}

func main() {
	err := config.ParseConfig()
	if err != nil {
		log.Fatal("解析配置文件出错:", err)
	}
	log.Printf("%+v\n", config.GlobalConfig)
	Start()
}
