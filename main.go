package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"

	"github.com/jamsa/hgap/config"
	"github.com/jamsa/hgap/inbound"
	"github.com/jamsa/hgap/outbound"
)

var help = `
  Usage: hgap [command]
  Commands:
    inbound - run as inbound server mode
    outbound - run as outbound server mode
`

func initLog(subcmd string, cfg *config.LogConfig) {
	//log.SetFormatter(&log.JSONFormatter{})

	writer, _ := rotatelogs.New(
		cfg.File+"-"+subcmd+".%Y%m%d%H%M.log",
		rotatelogs.WithLinkName(cfg.File+".log"),                                 // 生成软链，指向最新日志文件
		rotatelogs.WithMaxAge(time.Duration(cfg.MaxAge)*time.Second),             // 文件最大保存时间
		rotatelogs.WithRotationTime(time.Duration(cfg.RotationTime)*time.Second), // 日志切割时间间隔
	)
	//pathMap := lfshook.WriterMap{
	//logrus.InfoLevel:  writer,
	//logrus.PanicLevel: writer,
	//}
	/*hook =Hooks.Add(lfshook.NewHook(
		pathMap,
		&logrus.JSONFormatter{},
	))*/
	hook := lfshook.NewHook(writer, &log.TextFormatter{})
	//lfshook.NewHook()

	log.AddHook(hook)
}

func main() {
	flag.Parse()
	args := flag.Args()
	subcmd := ""
	if len(args) > 0 {
		subcmd = args[0]
		args = args[1:]
	}

	cfg, err := config.ParseConfig()
	if err != nil {
		log.Fatal("解析配置文出错", err)
	}
	log.Printf("%+v\n", cfg)

	initLog(subcmd, cfg.Log)

	switch subcmd {
	case "inbound":
		//inbound.Start()
		inb, err := inbound.New(cfg)
		if err != nil {
			log.Fatal("无法启动InBound服务", err)
		}
		inb.Start()
	case "outbound":
		//outbound.Start()
		outb, err := outbound.New(cfg)
		if err != nil {
			log.Fatal("无法启动OutBound服务", err)
		}
		outb.Start()
	default:
		fmt.Print(help)
		os.Exit(0)
	}
}
