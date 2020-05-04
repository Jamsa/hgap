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

	fsmon "github.com/jamsa/hgap/fsmon"
)

func processRequest(fileName string) {
	_, file := filepath.Split(fileName)
	reqID := strings.TrimSuffix(file, filepath.Ext(file))

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

func main() {
	log.Println("开始监视请求文件目录")
	fsmon.StartWatcher("in/req", processRequest)
}
