package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

func processRequest(reqID string) {
	//读取请求信息
	f, err := os.Open("in/req/" + reqID + ".req")
	if err != nil {
		log.Fatal("读取请求文件"+reqID+"出错", err)
	}
	defer f.Close()
	buf := bufio.NewReader(f)
	//for {
	req, err := http.ReadRequest(buf)
	/*if err == io.EOF {
		break
	}*/
	if err != nil && err != io.EOF {
		log.Fatal("error", err)
	}
	//}

	//输出从请求备份文件中恢复的请求信息
	content, err := httputil.DumpRequest(req, true)
	//log.Println("解析出来的请求信息：")
	//log.Println(string(content))

	//url := fmt.Sprintf("%s://%s%s", "http", "www.baidu.com", req.RequestURI)
	url := fmt.Sprintf("%s://%s%s", "http", "localhost:3005", req.RequestURI)
	log.Println(url)
	//
	body, err := ioutil.ReadAll(req.Body)
	//转发请求
	proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))
	proxyReq.Header = make(http.Header)
	for h, val := range req.Header {
		proxyReq.Header[h] = val
	}
	httpClient := &http.Client{}
	resp, err := httpClient.Do(proxyReq)

	//保存响应
	content, err = httputil.DumpResponse(resp, true)
	ioutil.WriteFile("out/resp/"+reqID+".resp", content, 0644)
	log.Println("写入响应文件:" + reqID)
}

func startWatcher() {
	watch, err := fsnotify.NewWatcher()
	watch.Add("in/req")
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
							go processRequest(chName)
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

func main() {
	log.Println("开始监视请求文件目录")
	startWatcher()
}
