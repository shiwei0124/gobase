// bytes_stream_test.go
package gobase

import (
	"fmt"
	"testing"
)

func Test_Write(t *testing.T) {
	bytesStream := NewBytesStreamW()
	bytesStream.WriteUint16(64)
	tmp := NewBytesStreamR(bytesStream.Data())
	n := tmp.ReadUint16()
	fmt.Println(n)

}
