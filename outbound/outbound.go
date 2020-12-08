package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
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
	reqID := strings.TrimSuffix(file, filepath.Ext(file))

	//读取请求信息
	//f, err := os.Open( config.GlobalConfig.InDirectory + "/" + reqID + ".req")
	var buf *bufio.Reader
	if config.GlobalConfig.TextTransfer {
		content, err := ioutil.ReadFile(filepath.Join(config.GlobalConfig.InDirectory, reqID+".req"))
		if err != nil {
			log.Println("读取请求文件", reqID, "出错", err)
			return
		}
		if config.GlobalConfig.TextTransfer {
			content, err = base64.StdEncoding.DecodeString(string(content))
			if err != nil {
				log.Println("解码文件", reqID, "出错", err)
				return
			}
		}
		buf = bufio.NewReader(strings.NewReader(string(content)))
	} else {
		f, err := os.Open(filepath.Join(config.GlobalConfig.InDirectory, reqID+".req"))
		if err != nil {
			log.Println("读取请求文件", reqID, "出错", err)
			return
		}
		defer f.Close()
		buf = bufio.NewReader(f)
	}

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
				//log.Println("#####:", h, "-----", val)
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
			//err = ioutil.WriteFile(config.GlobalConfig.OutDirectory+"/"+reqID+".resp", content, 0644)

			eof := "EOF" + reqID
			if config.GlobalConfig.TextTransfer {
				content = []byte(base64.StdEncoding.EncodeToString(content))
			}
			content = append(content, []byte(eof)...)

			err = ioutil.WriteFile(filepath.Join(config.GlobalConfig.OutDirectory, reqID+".resp"), content, 0644)
			if err != nil {
				log.Println("写入响应文件出错", err)
				return
			}

			/*
				noti, err := os.Create(filepath.Join(config.GlobalConfig.OutDirectory, reqID+".noti"))
				if err != nil {
					log.Println("写入请求文件出错", err)
					return
				}
				noti.Close()
				err = os.Rename(filepath.Join(config.GlobalConfig.OutDirectory, reqID+".tmp"), filepath.Join(config.GlobalConfig.OutDirectory, reqID+".resp"))
				if err != nil {
					log.Println("重命名响应文件出错", err)
					return
				}
			*/

			log.Println("写入响应文件完成:" + reqID)
			return
		}
	}
}

func main() {
	err := config.ParseConfig()
	if err != nil {
		log.Fatal("解析配置文出错", err)
	}
	log.Printf("%+v\n", config.GlobalConfig)
	log.Println("开始监视请求文件目录")
	//fsmon.StartWatcher(config.GlobalConfig.InDirectory, processRequest)
	fsmon.StartScan(config.GlobalConfig.InDirectory, processRequest)
}
