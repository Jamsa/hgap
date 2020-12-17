package packet

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"io"
	"math"
)

// MTU 最大传输单元
const MTU = 1024

// Packet 数据包分组
type Packet struct {
	ID     string //标识
	Length int    //总长
	Begin  int    //开始位置
	Size   int    //数据长度
	Data   []byte //数据
}

// Encode Packet编码
func (packet *Packet) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(packet)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decode Packet解码
func (packet *Packet) Decode(data []byte) error {
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(packet)
	if err != nil {
		return err
	}
	return nil
}

// Iterator Packet 迭代器
type Iterator struct {
	id      string
	size    int
	data    []byte
	current int
}

// HasNext 下一个
func (iter *Iterator) HasNext() bool {
	count := math.Ceil(float64(len(iter.data)) / float64(iter.size))
	//log.Printf("HasNext:%v,%v,%v,%v,%v", len(iter.data), iter.size, iter.current, count, iter.current+1 <= int(count))
	return iter.current+1 <= int(count)
}

// Next 下一个
func (iter *Iterator) Next() *Packet {
	begin := iter.current * iter.size
	end := (iter.current + 1) * iter.size
	length := len(iter.data)

	if begin >= length {
		return nil
	}
	if end > length {
		end = length
	}
	//log.Printf("%v,%v,%v", begin, end, length)
	iter.current = iter.current + 1
	result := Packet{
		ID:     iter.id,
		Length: length,
		Begin:  begin,
		Size:   end - begin, // iter.size
		Data:   iter.data[begin:end],
	}

	return &result
}

// NewIterator 新建迭代器
func NewIterator(id string, data []byte, size int) *Iterator {
	return &Iterator{
		id:      id,
		size:    1024, //分组长
		data:    data,
		current: 0,
	}
}

// FrameType 数据帧类型
type FrameType int32

// 数据帧类型定义
const (
	FrameTypeHELLO FrameType = 0
	FrameTypeDATA  FrameType = 1
)

// FrameMagic Frame头的MagicNumber
const FrameMagic uint32 = 0x123456

// Frame 数据帧，用于在Tcp等非固定分组大小的模式下传输数据
type Frame struct {
	FrameType FrameType // 帧类型
	Length    int32     // 数据长度
	Data      []byte    // 数据
}

// Write Frame编码
func (frame *Frame) Write(writer io.Writer) error {
	err := binary.Write(writer, binary.BigEndian, FrameMagic)
	err = binary.Write(writer, binary.BigEndian, frame.Length)
	err = binary.Write(writer, binary.BigEndian, frame.FrameType)
	err = binary.Write(writer, binary.BigEndian, frame.Data)
	return err
}

func splitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	//FrameMagic+FrameType+Length 3个int32的长度
	if !atEOF &&
		len(data) > 4*3 &&
		binary.BigEndian.Uint32(data[:4]) == FrameMagic {
		var t, l int32
		err := binary.Read(bytes.NewReader(data[4:8]), binary.BigEndian, &t)
		err = binary.Read(bytes.NewReader(data[8:12]), binary.BigEndian, &l)
		if err != nil {
			//TODO 读取错误时，adv和token该返回什么值
			return 0, nil, err
		}
		end := 4*3 + l
		//消费end长的数据，返回从第4位开始的完整Frame数据
		return int(end), data[4:end], nil
	}
	return
}

// Reader Frame解码
func (frame *Frame) Read(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	scanner.Split(splitFunc)
	s := scanner.Scan()
	if s {
		data := scanner.Bytes()
		var t, l int32
		err := binary.Read(bytes.NewReader(data[:4]), binary.BigEndian, &t)
		err = binary.Read(bytes.NewReader(data[4:8]), binary.BigEndian, &l)
		if err != nil {
			return err
		}
		frame.FrameType = FrameType(t)
		frame.Length = l
		frame.Data = data[8:]
		return nil
	}
	return nil
}
