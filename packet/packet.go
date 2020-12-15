package packet

import (
	"math"
)

// Packet 数据包分组
type Packet struct {
	ID     string //标识
	Length int    //总长
	Begin  int    //开始位置
	Size   int    //数据长度
	Data   []byte //数据
}

/*
// Encode 转 []byte
func (packet *Packet) Encode() []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, packet.ID)
	binary.Write(buf, binary.LittleEndian, packet.Length)
	binary.Write(buf, binary.LittleEndian, packet.Begin)
	binary.Write(buf, binary.LittleEndian, packet.Size)
	binary.Write(buf, binary.LittleEndian, packet.Data)

	return buf.Bytes()
}

// Decode 转 Packet
func (packet *Packet) Decode([]byte) error {
	var buf bytes.Buffer

	return err
}
*/

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
		Size:   iter.size,
		Data:   iter.data[begin:end],
	}

	return &result
}

// NewIterator 新建迭代器
func NewIterator(id string, data []byte) *Iterator {
	return &Iterator{
		id:      id,
		size:    1024, //分组长
		data:    data,
		current: 0,
	}
}
