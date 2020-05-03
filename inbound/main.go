package main

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	uuid "github.com/satori/go.uuid"
)

var reqs = make(map[string]chan struct{})

func saveRequest(name string, timeout int) {

}

func writeResponse() {

}

func startWatcher() {
	watch, err := fsnotify.NewWatcher()
	watch.Add("out/resp")
	if err != nil {
		log.Fatal("创建watcher失败", err)
	}
	for {
		select {
		case ev := <-watch.Events:
			{
				if ev.Op&fsnotify.Create == fsnotify.Create {
					log.Println("创建文件: ", ev.Name)

					if fi, err := os.Stat(ev.Name); err == nil {
						if !fi.IsDir() {
							_, file := filepath.Split(ev.Name)
							chName := strings.TrimSuffix(file, filepath.Ext(file))
							ch, ok := reqs[chName]
							if ok {
								log.Println("发送响应文件通知:" + chName)
								ch <- struct{}{}
							} else {
								log.Println("文件响应通道不存在:" + chName)
							}
						}
					}
				}
				if ev.Op&fsnotify.Write == fsnotify.Write {
					log.Println("写入文件: ", ev.Name)

				}
				if ev.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("删除文件: ", ev.Name)

				}
				if ev.Op&fsnotify.Rename == fsnotify.Rename {
					log.Println("重命名文件: ", ev.Name)
				}
				if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
					log.Println("修改权限: ", ev.Name)
				}
			}
		case err := <-watch.Errors:
			{
				log.Println("error : ", err)
				return
			}
		}
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	content, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Fatal("error:", err)
	}
	uid, err := uuid.NewV4()
	if err != nil {
		log.Fatal("error", err)
	}
	reqID := uid.String()

	log.Println("保存请求:" + reqID)
	//log.Println(string(content))

	ioutil.WriteFile("in/req/"+reqID+".req", content, 0644)
	finish := make(chan struct{})
	reqs[reqID] = finish
	//最长20秒超时
	timeout := time.NewTicker(20000 * time.Millisecond)

	cleanUp := func() {
		timeout.Stop()
		delete(reqs, reqID)
		// TODO: 删除文件
		close(finish)
	}

	writeResp := func() {
		//读取响应
		log.Println("读取响应:" + reqID)
		f, err := os.Open("out/resp/" + reqID + ".resp")
		if err != nil {
			log.Fatal("error", err)
		}
		defer f.Close()
		buf := bufio.NewReader(f)
		resp, err := http.ReadResponse(buf, r)

		/*resp.Body.Close()
		b := new(bytes.Buffer)
		io.Copy(b, resp.Body)
		resp.Body.Close()*/
		for h, val := range resp.Header {
			w.Header().Set(h, val[0])
		}
		b := new(bytes.Buffer)
		io.Copy(b, resp.Body)
		resp.Body.Close()
		//输出从响应备份文件中恢复的响应内容
		w.Write(b.Bytes())
	}

	select {
	case <-finish:
		log.Println("获取响应")
		writeResp()
	case <-timeout.C:
		log.Println("请求处理超时")

	}
	cleanUp()
}

func main() {
	go startWatcher()
	http.HandleFunc("/", index)
	log.Println("开始监听9090...")
	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("监听出错: ", err)
	}
}
