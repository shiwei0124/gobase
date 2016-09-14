package gobase

import (
	"fmt"
	"testing"
)

func Test_ParseRedis(b *testing.T) {
	data := "*2\r\n$3\r\nfoo\r\n$6\r\nbarddd\r\n"
	//data = "*3\r\n:1\r\n:-2\r\n:3\r\n"
	data = "*2\r\n*3\r\n:1\r\n:2\r\n:3\r\n*2\r\n+Foo\r\n-Bar\r\n"
	//data = "*2\r\n+Foo\r\n-Bar\r\n"
	data2 := make([]byte, 0)
	data2 = append(data2, data...)
	fmt.Println("ddddd")
	n := 1000000
	for i := 0; i < n; i++ {
		cmd := NewRedisCmd([]byte(data))
		if _, err := cmd.ParseRequest(); err != nil {
			fmt.Println(err)
		} else {
			//printRESP(respData)
			//fmt.Println(type(a))
		}
	}
	fmt.Println("aaa")
}

func Test_ParseLargeRESP(b *testing.T) {

}
