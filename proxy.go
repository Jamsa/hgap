package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
)

// Serve 服务
func Serve(w http.ResponseWriter, req *http.Request) {
	buf, err := httputil.DumpRequest(req, true)
	if err != nil {
		return
	}
	fmt.Println(string(buf))

}

func main() {
	http.HandleFunc("/", Serve)
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
