package transfer

import (
	"encoding/base64"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// FileTransfer 文件传输
type FileTransfer struct {
	ITransfer
	*Transfer
	path     string //文件保存目录
	fileExt  string //文件扩展名
	keepFile bool   //保留传输的文件
}

// Send 发送文件
func (transfer *FileTransfer) Send(reqID string, data []byte) {
	eof := "EOF" + reqID
	content := data
	if transfer.textTransfer {
		content = []byte(base64.StdEncoding.EncodeToString(data))
	}
	content = append(content, []byte(eof)...)

	err := ioutil.WriteFile(filepath.Join(transfer.path, reqID)+transfer.fileExt, content, 0644)
	if err != nil {
		log.Println("写入请求文件出错", err)
		return
	}
}

// Remove 删除文件
func (transfer *FileTransfer) Remove(reqID string) {
	if !transfer.keepFile {
		log.Println("删除传输的文件" + reqID)
		os.Remove(filepath.Join(transfer.path, reqID) + transfer.fileExt)
	}
}
