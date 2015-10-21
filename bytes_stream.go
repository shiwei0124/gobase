// bytes_data.go
package gobase

import (
	"fmt"
	"encoding/binary"
)

func main() {
	fmt.Println("Hello World!")
}

type BytesStream struct {
	data []byte
	pos int
}

func NewBytesStream(data []byte) * BytesStream {
	bytesStream := &BytesStream {
		data : data,
		pos : 0,
	}
	return bytesStream
}
 
func (s * BytesStream) ReadByte() byte {
	res := s.data[s.pos]
	s.pos += 1
	return res
}

func (s * BytesStream) ReadUint16() uint16 {
	res := binary.BigEndian.Uint16(s.data[s.pos:s.pos+2])
	s.pos += 2
	return res
}

func (s * BytesStream) ReadUint32() uint32 {
	res := binary.BigEndian.Uint32(s.data[s.pos:s.pos+4])
	s.pos += 4
	return res
}

func (s * BytesStream) ReadUint64() uint64 {
	res := binary.BigEndian.Uint64(s.data[s.pos:s.pos+8])
	s.pos += 8
	return res
}

func (s * BytesStream) ReadString(length int) string {
	res := string(s.data[s.pos:s.pos+length])
	s.pos += length
	return res
}

func (s * BytesStream) ReadBytes(length int) []byte {
	res := s.data[s.pos:s.pos+length]	
	s.pos += length
	return res
}

func (s *BytesStream) WriteByte(data byte) {
	
}

func (s *BytesStream) WriteUint16(data uint16) {
	
}

func (s *BytesStream) WriteUint32(data uint32) {
	
}

func (s *BytesStream) WriteUint64(data uint64) {
	
}

func (s *BytesStream) WriteString(data string) {
	
}

func (s *BytesStream) WriteBytes(data []byte) {
	
}

