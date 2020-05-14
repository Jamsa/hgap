package fsmon

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/jamsa/hgap/config"
)

// StartWatcher 启动目录变化监控
func StartWatcher(path string, createdHandler func(fileName string)) {
	watch, err := fsnotify.NewWatcher()
	watch.Add(path)
	if err != nil {
		log.Fatal("目录监控出错", err)
	}
	log.Println("监视目录" + path)
	for {
		select {
		case ev := <-watch.Events:
			{
				if ev.Op&fsnotify.Create == fsnotify.Create {
					log.Println("创建文件: ", ev.Name)
					if fi, err := os.Stat(ev.Name); err == nil {
						if !fi.IsDir() {
							go createdHandler(ev.Name)
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
				log.Println("文件监控出错: ", err)
				return
			}
		}
	}
}

func checkFile(filename string, eof string, eoflen int) (result bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("checkFile未知错误", r)
			debug.PrintStack()
			result = false
		}
	}()

	file, err := os.OpenFile(filename, os.O_RDWR, 0644)
	if err != nil {
		log.Println("checkFile文件无法打开", err)
		return false
	}
	defer file.Close()

	buf := make([]byte, eoflen)

	stat, err := file.Stat() //os.Stat(filename)
	if err != nil {
		log.Println("checkFile无法获取文件信息", nil)
		return false
	}
	start := stat.Size() - int64(eoflen)
	if start < 0 {
		log.Println("checkFile文件大小不匹配", start)
		return false
	}
	_, err = file.ReadAt(buf, start)
	if err == nil && string(buf) == eof {
		log.Println("checkFile文件结束内容匹配", string(buf))
		file.Seek(0, 0)
		err = file.Truncate(start)
		if err != nil {
			log.Println("checkFile truncate file出错", err, start)
		}
		file.Sync()

		return true
	}
	log.Println("checkFile文件结束内容不匹配", err)
	return false
}

// WaitForFile 等侍文件就绪或超时
func WaitForFile(filename string) error {
	start := time.Now()
	_, file := filepath.Split(filename)
	reqID := strings.TrimSuffix(file, filepath.Ext(file))
	eof := "EOF" + reqID
	eoflen := len([]byte(eof))
	for {
		if time.Since(start) > time.Duration(config.GlobalConfig.Timeout) {
			return errors.New("文件处理超时")
		}

		result := checkFile(filename, eof, eoflen)
		if result {
			return nil
		}
		time.Sleep(time.Duration(config.GlobalConfig.FileCheckInterval) * time.Millisecond)
	}
}
