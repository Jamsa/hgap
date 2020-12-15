package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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
