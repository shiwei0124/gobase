// bytes_data.go
package gobase

import (
	"encoding/binary"
)

type BytesStream struct {
	data []byte
	pos  int
}

// unpack data
func NewBytesStreamR(data []byte) *BytesStream {
	bytesStream := &BytesStream{
		data: data,
		pos:  0,
	}
	return bytesStream
}

func (s *BytesStream) Data() []byte {
	return s.data
}

func (s *BytesStream) ReadByte() byte {
	res := s.data[s.pos]
	s.pos += 1
	return res
}

func (s *BytesStream) ReadUint16() uint16 {
	res := binary.BigEndian.Uint16(s.data[s.pos : s.pos+2])
	s.pos += 2
	return res
}

func (s *BytesStream) ReadUint32() uint32 {
	res := binary.BigEndian.Uint32(s.data[s.pos : s.pos+4])
	s.pos += 4
	return res
}

func (s *BytesStream) ReadUint64() uint64 {
	res := binary.BigEndian.Uint64(s.data[s.pos : s.pos+8])
	s.pos += 8
	return res
}

func (s *BytesStream) ReadString(length int) string {
	res := string(s.data[s.pos : s.pos+length])
	s.pos += length
	return res
}

func (s *BytesStream) ReadBytes(length int) []byte {
	res := s.data[s.pos : s.pos+length]
	s.pos += length
	return res
}

// pack data
func NewBytesStreamW() *BytesStream {
	bytesStream := &BytesStream{
		data: make([]byte, 0),
		pos:  0,
	}
	return bytesStream
}

func (s *BytesStream) WriteByte(data byte) {
	s.data = append(s.data, data)
}

func (s *BytesStream) WriteUint16(data uint16) {
	tmp := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp, data)
	s.data = append(s.data, tmp...)
}

func (s *BytesStream) WriteUint32(data uint32) {
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, data)
	s.data = append(s.data, tmp...)
}

func (s *BytesStream) WriteUint64(data uint64) {
	tmp := make([]byte, 8)
	binary.BigEndian.PutUint64(tmp, data)
	s.data = append(s.data, tmp...)
}

func (s *BytesStream) WriteString(data string) {
	s.data = append(s.data, []byte(data)...)
}

func (s *BytesStream) WriteBytes(data []byte) {
	s.data = append(s.data, data...)
}
