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

	err := config.ParseConfig()
	if err != nil {
		log.Fatal("解析配置文出错", err)
	}
	log.Printf("%+v\n", config.GlobalConfig)

	switch subcmd {
	case "inbound":
		inbound.Start()
	case "outbound":
		outbound.Start()
	default:
		fmt.Print(help)
		os.Exit(0)
	}
}
