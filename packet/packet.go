package packet

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
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
		size:    size, //1024, //分组长
		data:    data,
		current: 0,
	}
}

// FrameType 数据帧类型
type FrameType int32

// 数据帧类型定义
const (
	FrameTypeCLOSE FrameType = 0
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

// Encode Frame编码
func (frame *Frame) Encode() ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.BigEndian, FrameMagic)
	err = binary.Write(&buf, binary.BigEndian, frame.Length)
	err = binary.Write(&buf, binary.BigEndian, frame.FrameType)
	_, err = buf.Write(frame.Data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode Frame解码
func (frame *Frame) Decode(data []byte) error {
	var t, l int32
	err := binary.Read(bytes.NewReader(data[:4]), binary.BigEndian, &t)
	err = binary.Read(bytes.NewReader(data[4:8]), binary.BigEndian, &l)
	if err != nil {
		return err
	}
	frame.FrameType = FrameType(t)
	frame.Length = l
	frame.Data = data[8:]
	//log.Printf("解码数据帧:%v,%v", frame.FrameType, frame.Length)
	return nil
}
