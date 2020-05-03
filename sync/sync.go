package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

type Config struct {
	IgnoreNamePatterns string            `json:"ignoreNamePatterns"`
	SyncRename         bool              `json:"syncRename"`
	SyncRemove         bool              `json:"syncRemove"`
	SyncWrite          bool              `json:"syncWrite"`
	DirMapping         map[string]string `json:"dirMapping"`
}

var dirMapping map[string]string

func main() {
	var cfg string
	var err error
	cfg = "sync.json"
	if len(os.Args) > 1 {
		if cfg, err = filepath.Abs(os.Args[1]); err != nil {
			log.Fatal("读取配置文件出错:", err)
		}
	}

	jsonFile, err := os.Open(cfg)
	if err != nil {
		log.Fatal("读取配置文件", cfg, "出错:", err)
	}

	log.Printf("读取sync.json")
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatal("读取json配置出错:", err)
	}

	err = json.Unmarshal([]byte(byteValue), &dirMapping)
	if err != nil {
		log.Fatal("解析json出错:", err)
	}

	watch, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("创建watcher失败", err)
	}

	defer watch.Close()
	workDir := filepath.Dir(jsonFile.Name())

	err = os.Chdir(workDir)
	if err != nil {
		log.Fatal("无法切换至json文件目录", err)
	} else {
		log.Println("切换至工作目录", workDir)
	}
	if !checkDirMapping() {
		os.Exit(1)
	}
	for k, _ := range dirMapping {
		var absPath string
		var fi os.FileInfo
		var err error
		if absPath, err = filepath.Abs(k); err != nil {
			log.Println("无法获取绝对路径", k, err)
			continue
		}
		if fi, err = os.Stat(absPath); err == nil && fi.IsDir() {
			watchDir(absPath, watch)
		} else {
			log.Println("无法监控", absPath, err)
		}
	}
	//watchDir(".", watch)
	go sync(watch)
	select {}
}

// 检查映射关系
func checkDirMapping() bool {
	for k, _ := range dirMapping {
		fromPath, fe := filepath.Abs(k)
		if fe == nil {
			for _, v := range dirMapping {
				toPath, te := filepath.Abs(v)
				if te == nil && strings.HasPrefix(toPath, fromPath) {
					log.Println("路径存在包含关系", fromPath, ":", toPath)
					return false
				}
			}
		}
	}
	return true
}

// 监控目录
func watchDir(dir string, watch *fsnotify.Watcher) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			var absPath string
			var err error
			if absPath, err = filepath.Abs(path); err != nil {
				return err
			}
			if err = watch.Add(absPath); err != nil {
				return err
			}
			log.Println("监控目录:", absPath)
		}
		return nil
	})
}

func getMappingPath(name string) (string, error) {
	safePath := "/tmp/nil.nil"
	for k, v := range dirMapping {
		var from, to string
		var err error
		if from, err = filepath.Abs(k); err != nil {
			return safePath, err
		}
		if to, err = filepath.Abs(v); err != nil {
			return safePath, err
		}
		if strings.HasPrefix(name, from) {
			r := strings.NewReplacer(from, to)
			path := r.Replace(name)
			return path, nil
		}
	}
	return safePath, errors.New(fmt.Sprintf("无法找到 %s的映射关系", name))
}

func syncRemove(name string) error {
	if strings.HasSuffix(name, "___") {
		return nil
	}
	var path string
	var err error
	if path, err = getMappingPath(name); err != nil {
		return err
	}

	if err = os.RemoveAll(path); err != nil {
		return errors.WithMessagef(err, "删除 %s 匹配的 %s 出错", name, path)
	}
	log.Printf("删除 %s匹配的 %s ", name, path)
	return nil
}

func syncCreateDir(name string) error {
	if strings.HasSuffix(name, "___") {
		return nil
	}
	var path string
	var srcInfo os.FileInfo
	var err error
	if path, err = getMappingPath(name); err != nil {
		return err
	}

	if srcInfo, err = os.Stat(name); err != nil {
		return err
	}
	if !srcInfo.IsDir() {
		return errors.New(fmt.Sprintf("%s不是目录", name))
	}

	if err = os.Mkdir(path, srcInfo.Mode()); err != nil {
		return errors.WithMessagef(err, "创建 %s 匹配的 %s 目录出错", name, path)
	}
	log.Printf("创建 %s匹配的 %s 目录", name, path)
	return nil
}

func syncWrite(name string) error {
	//TODO 延时3秒模拟同步
	time.Sleep(3 * time.Second)
	if strings.HasSuffix(name, "___") {
		return nil
	}
	var path string
	var err error
	var srcInfo, srcDirInfo os.FileInfo
	var srcfd, dstfd *os.File

	if srcInfo, err = os.Stat(name); err != nil {
		return err
	}

	if srcDirInfo, err = os.Stat(filepath.Dir(srcInfo.Name())); err != nil {
		return err
	}

	if path, err = getMappingPath(name); err != nil {
		return err
	}

	os.MkdirAll(filepath.Dir(path), srcDirInfo.Mode())

	if srcfd, err = os.Open(name); err != nil {
		return err
	}

	defer srcfd.Close()

	//没有则创建，有则覆盖
	if dstfd, err = os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, srcInfo.Mode()); err != nil {
		return err
	}

	defer dstfd.Close()

	if _, err = io.Copy(dstfd, srcfd); err != nil {
		errors.WithMessagef(err, "复制文件 %s 至 %s 出错", name, path)
	}
	log.Printf("复制文件 %s 至 %s", name, path)
	return nil
}

// 同步目录
func sync(watch *fsnotify.Watcher) {
	for {
		select {
		case ev := <-watch.Events:
			{
				if ev.Op&fsnotify.Create == fsnotify.Create {
					//log.Println("创建文件: ", ev.Name)

					if fi, err := os.Stat(ev.Name); err == nil {
						if fi.IsDir() {
							watchDir(ev.Name, watch)
							//watch.Add(ev.Name)
							//log.Println("添加监控目录 : ", ev.Name);
						} else {
							if err = syncWrite(ev.Name); err != nil {
								log.Println(err.Error())
							}
						}
					}
				}
				if ev.Op&fsnotify.Write == fsnotify.Write {
					//log.Println("写入文件: ", ev.Name);
					if err := syncWrite(ev.Name); err != nil {
						log.Println(err.Error())
					}
				}
				if ev.Op&fsnotify.Remove == fsnotify.Remove {
					//log.Println("删除文件: ", ev.Name);
					if err := watch.Remove(ev.Name); err != nil {
						//log.Println("aaa",err)
					} else {
						log.Println("移除监控目录: ", ev.Name)
					}
					if err := syncRemove(ev.Name); err != nil {
						log.Println(err.Error())
					}
				}
				if ev.Op&fsnotify.Rename == fsnotify.Rename {
					//log.Println("重命名文件: ", ev.Name);

					if err := watch.Remove(ev.Name); err != nil {
						//log.Println("aaa",err)
					} else {
						log.Println("移除监控目录: ", ev.Name)
					}
					//if err := syncRemove(ev.Name); err != nil {
					//	log.Println(err.Error())
					//}
				}
				if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
					//log.Println("修改权限: ", ev.Name);
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
