package fsmon

import (
	"log"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
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
						if !fi.IsDir() && !strings.HasSuffix(ev.Name, ".tmp") {
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
