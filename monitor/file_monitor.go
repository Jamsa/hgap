package monitor

import (
	"encoding/base64"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// FileMonitor 文件系统监视
type FileMonitor struct {
	IMonitor
	*Monitor
	path string //监视目录
	//suffix        string           //文件后续
	scanInterval  int              //扫描频度(ms)
	timeout       int              //等侍文件就绪的超时时间(ms)
	checkInterval int              //检查频度(ms)
	fileExt       string           //文件扩展名
	lastFiles     map[string]int64 //最后一次扫描的目录文件清单
	keepFile      bool             //保留接收到的文件
}

// TODO 增加cleanUp定时清理目录下的垃圾文件

// Start 启动监视
func (monitor *FileMonitor) Start(onReady OnReady) {
	monitor.onReady = onReady
	log.Println("开始监视文件目录", monitor.path)
	for {
		//start := time.Now()
		path := monitor.path
		lastFiles := monitor.lastFiles
		newFiles := make(map[string]int64)
		files, err := ioutil.ReadDir(path)
		if err != nil {
			log.Error("获取目录文件列表出错", path, err)
			continue
		}
		for i := 0; i < len(files); i++ {
			file := files[i]
			fileSize := file.Size()
			fileName := file.Name()
			if !file.IsDir() && fileSize > 0 {
				newFiles[fileName] = fileSize
				_, ok := lastFiles[fileName]
				if ok {

				} else {
					log.Println("新的文件", fileName)
					go monitor.createHandler(fileName)
				}
			}
		}
		monitor.lastFiles = newFiles
		time.Sleep(time.Duration(monitor.scanInterval) * time.Millisecond)
	}
}

// Remove 删除数据
func (monitor *FileMonitor) Remove(reqID string) {
	if !monitor.keepFile {
		log.Println("删除监控的文件" + reqID)
		os.Remove(filepath.Join(monitor.path, reqID) + monitor.fileExt)
	}
}

// Read 读取数据
func (monitor *FileMonitor) Read(reqID string) ([]byte, error) {
	fileName := reqID + monitor.fileExt
	return monitor.readFile(fileName)
}

// DebugTimeout 超时诊断
func (monitor *FileMonitor) DebugTimeout(reqID string) {
	log.Debug("FileMonitor DebugTimeout(NOP)")
}

// 文件创建
func (monitor *FileMonitor) createHandler(fileName string) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("处理请求文件", fileName, "出错", r)
		}
	}()

	if err := monitor.waitForFile(fileName); err != nil {
		log.Error("等侍文件就绪时出错", err)
		return
	}

	/*buf, err := monitor.readFile(fileName)
	if err != nil {
		log.Println("读取请求文件时出错", err)
		return
	}
	if !monitor.keepFile {
		os.Remove(filepath.Join(monitor.path, fileName))
	}*/

	reqID := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	monitor.onReady(reqID) //, buf)
}

// 读取文件
func (monitor *FileMonitor) readFile(fileName string) ([]byte, error) {
	fullpath := filepath.Join(monitor.path, fileName)
	content, err := ioutil.ReadFile(fullpath)
	if err != nil {
		//log.Println("读取请求文件", fileName, "出错", err)
		return nil, err
	}
	if monitor.textTransfer {
		content, err = base64.StdEncoding.DecodeString(string(content))
		if err != nil {
			//log.Println("解码文件", fileName, "出错", err)
			return nil, err
		}
	}
	return content, nil
}

// 等侍文件就绪
func (monitor *FileMonitor) waitForFile(fileName string) error {
	start := time.Now()
	fullpath := filepath.Join(monitor.path, fileName)
	reqID := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	eof := "EOF" + reqID

	for {
		if time.Since(start) > time.Duration(monitor.timeout)*time.Millisecond {
			return errors.New("文件处理超时")
		}

		result := checkFile(fullpath, eof)
		if result {
			return nil
		}
		time.Sleep(time.Duration(monitor.checkInterval) * time.Millisecond)
	}
}

//检查文件完整性
func checkFile(filename string, eof string) (result bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("checkFile未知错误", r)
			debug.PrintStack()
			result = false
		}
	}()
	eoflen := len([]byte(eof))

	if runtime.GOOS == "windows" {
		if err := os.Chmod(filename, 0600); err != nil {
			log.Error("checkFile文件无法改为0600", err)
			return false
		}
	}

	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		log.Error("checkFile文件无法打开", err)
		return false
	}
	defer file.Close()

	buf := make([]byte, eoflen)

	stat, err := file.Stat() //os.Stat(filename)
	if err != nil {
		log.Error("checkFile无法获取文件信息", filename, nil)
		return false
	}
	start := stat.Size() - int64(eoflen)
	if start < 0 {
		log.Error("checkFile文件大小不匹配", filename, start)
		return false
	}
	_, err = file.ReadAt(buf, start)
	if err == nil && string(buf) == eof {
		log.Debug("checkFile文件结束内容匹配", filename, string(buf))
		file.Seek(0, 0)
		err = file.Truncate(start)
		if err != nil {
			log.Error("checkFile truncate file出错", err, start)
		}
		file.Sync()

		return true
	}
	log.Error("checkFile文件结束内容不匹配", err)
	return false
}
