package main

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
	uuid "github.com/satori/go.uuid"
)

// ChannelWrapper 包装
//type ChannelWrapper chan interface{}

// UnWrapper 解包
/*func (ch ChannelWrapper) UnWrapper() chan interface{} {
	return ch
}*/

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
	if config.GlobalConfig.TextTransfer {
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
		if config.GlobalConfig.TextTransfer {
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

func main() {
	err := config.ParseConfig()
	if err != nil {
		log.Fatal("解析配置文件出错:", err)
	}
	log.Printf("%+v\n", config.GlobalConfig)
	//go fsmon.StartWatcher(config.GlobalConfig.OutDirectory, fileChangeHandle)
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
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal("监听出错: ", err)
	}
}
