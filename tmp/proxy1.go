package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"strings"
	"time"
)

func mainP1() {
	start := time.Now()

	println(time.Duration(30000))
	println(time.Duration(30000) * time.Millisecond)
	println(time.Duration(30) * time.Second)
	//time.Since(start) > time.Duration(config.GlobalConfig.Timeout)*time.Millisecond {

	time.Sleep(time.Duration(2) * time.Second)
	println(time.Since(start).Milliseconds())
	//println(time.Since(start).Milliseconds())
	if time.Since(start) > time.Duration(2000)*time.Millisecond {
		println("######")
	}
}

func main1() {
	l, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Panic(err)
	}
	for {
		client, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}

		go handleClientRequest(client)
	}
}

func handleClientRequest(client net.Conn) {
	if client == nil {
		return
	}

	defer client.Close()
	var b [1024]byte
	n, err := client.Read(b[:])
	if err != nil {
		log.Println(err)
		return
	}
	var method, host, address string
	fmt.Sscanf(string(b[:bytes.IndexByte(b[:], '\n')]),
		"%s%s",
		&method,
		&host)
	hostPortURL, err := url.Parse(host)
	if err != nil {
		log.Println(err)
		return
	}
	if hostPortURL.Opaque == "443" {
		address = hostPortURL.Scheme + ":443"
	} else {
		if strings.Index(hostPortURL.Host, ":") == -1 {
			address = hostPortURL.Host + ":80"
		} else {
			address = hostPortURL.Host
		}
	}

	server, err := net.Dial("tcp", address)
	if err != nil {
		log.Println(err)
		return
	}

	if method == "CONNECT" {
		fmt.Fprint(client, "HTTP/1.1 200 Connection established\r\n")
	} else {
		server.Write(b[:n])
	}

	go io.Copy(server, client)
	io.Copy(client, server)
}
