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
	cmd := NewRedisCmd([]byte(data))
	if respData, err := cmd.ParseRequest(); err != nil {
		fmt.Println(err)
	} else {
		printRESP(respData)
		//fmt.Println(type(a))
	}
}
