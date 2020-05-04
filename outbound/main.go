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

	"github.com/jamsa/hgap/config"
	fsmon "github.com/jamsa/hgap/fsmon"
)

func processRequest(fileName string) {
	_, file := filepath.Split(fileName)
	reqID := strings.TrimSuffix(file, filepath.Ext(file))

	//读取请求信息
	f, err := os.Open(config.GlobalConfig.InDirectory + "/" + reqID + ".req")
	if err != nil {
		log.Println("读取请求文件", reqID, "出错", err)
		return
	}
	defer f.Close()
	buf := bufio.NewReader(f)
	req, err := http.ReadRequest(buf)
	if err != nil && err != io.EOF {
		log.Println("读取请求信息出错", err)
		return
	}

	//输出从请求备份文件中恢复的请求信息
	//content, err := httputil.DumpRequest(req, true)
	//log.Println("解析出来的请求信息：")
	//log.Println(string(content))

	//url := fmt.Sprintf("%s://%s%s", "http", "www.baidu.com", req.RequestURI)
	//url := fmt.Sprintf("%s://%s%s", "http", "localhost:3005", req.RequestURI)
	for k, v := range config.GlobalConfig.URLMapping {
		if strings.HasPrefix(req.RequestURI, k) {
			url := strings.Replace(req.RequestURI, k, v, 1)
			log.Println("URL重写:", req.RequestURI, "  -->  ", url)
			//
			body, err := ioutil.ReadAll(req.Body)
			if err != nil {
				log.Println("读取请求Body出错", err)
				return
			}
			//转发请求
			proxyReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))
			if err != nil {
				log.Println("构造请求对象出错", err)
				return
			}
			proxyReq.Header = make(http.Header)
			for h, val := range req.Header {
				proxyReq.Header[h] = val
			}

			httpClient := &http.Client{}
			resp, err := httpClient.Do(proxyReq)
			if err != nil {
				log.Println("执行请求时出错", err)
				return
			}

			//保存响应
			content, err := httputil.DumpResponse(resp, true)
			if err != nil {
				log.Println("Dump响应信息出错", err)
				return
			}
			err = ioutil.WriteFile(config.GlobalConfig.OutDirectory+"/"+reqID+".resp", content, 0644)
			if err != nil {
				log.Println("写入响应文件出错", err)
				return
			}
			log.Println("写入响应文件:" + reqID)
			return
		}
	}
}

func main() {
	err := config.ParseConfig()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("开始监视请求文件目录")
	fsmon.StartWatcher(config.GlobalConfig.InDirectory, processRequest)
}
