// redis_parser.go
package gobase

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
)

type RESP_SIMPLE_STRING string
type RESP_ERROR string
type RESP_INTEGER int64
type RESP_BULK_STRING string
type RESP_ARRAY []interface{}

var (
	okReply   RESP_SIMPLE_STRING = "OK"
	pongReply RESP_SIMPLE_STRING = "PONG"
)

var ErrUnexpectedRESPEOF = errors.New("unexpected RESP EOF")
var ErrBufferFullRESP = errors.New("buffered full RESP")

type RedisCmd struct {
	data          []byte
	r             *bytes.Reader
	br            *bufio.Reader
	neededDataLen int
	//respParseType RESPParseType
}

func NewRedisCmd(data []byte) *RedisCmd {
	return &RedisCmd{
		data:          data,
		neededDataLen: 0,
	}
}

func (c *RedisCmd) Data() []byte {
	return c.data
}

func (c *RedisCmd) String() string {
	return string(c.data)
}

func (c *RedisCmd) ParseRequest() (interface{}, error) {
	if object, err := c.parseRESP(); err != nil {
		if err == ErrUnexpectedRESPEOF || err == ErrBufferFullRESP {
			return nil, err
		} else {
			return nil, errors.New(fmt.Sprintf("parse redis request failed, err: %s", err.Error()))
		}
	} else {
		return object, err
	}
}

func (c *RedisCmd) ParseResponse() (interface{}, error) {
	if object, err := c.parseRESP(); err != nil {
		if err == ErrUnexpectedRESPEOF || err == ErrBufferFullRESP {
			return nil, err
		} else {
			return nil, errors.New(fmt.Sprintf("parse redis response failed, err: %s", err.Error()))
		}
	} else {
		return object, err
	}
}

func (c *RedisCmd) parseRESP() (interface{}, error) {
	//reader := bytes.NewReader(c.data)
	//bufio 的 NewReader Size默认值比reader大的话，会reset多损耗性能
	//br := bufio.NewReaderSize(reader, reader.Len())
	if c.r == nil {
		c.r = bytes.NewReader(c.data)
		c.br = bufio.NewReaderSize(c.r, c.r.Len())
	} else {
		c.r.Reset(c.data)
		c.br.Reset(c.r)
	}
	if resp, neededDataLen, err := parseRESP(c.br); err != nil {
		if err == ErrUnexpectedRESPEOF || err == ErrBufferFullRESP {
			//数据还没收完，则重新copy一份内存保存数据，避免使用原先的[]byte导致覆盖
			dataTmp := make([]byte, 0)
			dataTmp = append(dataTmp, c.data...)
			c.data = dataTmp
		}
		c.neededDataLen = neededDataLen
		return nil, err
	} else {
		c.neededDataLen = 0
		return resp, nil
	}
}

func (c *RedisCmd) AppendData(extraData []byte) bool {
	c.data = append(c.data, extraData...)
	extraDataLen := len(extraData)
	c.neededDataLen -= extraDataLen
	if c.neededDataLen <= 0 {
		c.neededDataLen = 0
		return true
	} else {
		//数据仍然不够
		return false
	}
}

func readLine(br *bufio.Reader) ([]byte, error) {
	p, err := br.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		return nil, ErrBufferFullRESP
	}
	if err != nil {
		return nil, err
	}
	i := len(p) - 2
	if i < 0 || p[i] != '\r' {
		return nil, errors.New("bad cmd line terminator")
	}
	return p[:i], nil
}

// parseLen parses bulk string and array lengths.
func parseLen(p []byte) (int, error) {
	if len(p) == 0 {
		return -1, errors.New("malformed length")
	}

	if p[0] == '-' && len(p) == 2 && p[1] == '1' {
		// handle $-1 and $-1 null replies.
		return -1, nil
	}

	var n int
	for _, b := range p {
		n *= 10
		if b < '0' || b > '9' {
			return -1, errors.New("illegal bytes in length")
		}
		n += int(b - '0')
	}

	return n, nil
}

func parseSimpleString(line []byte) (RESP_SIMPLE_STRING, error) {
	switch {
	case len(line) == 2 && line[0] == 'O' && line[1] == 'K':
		// Avoid allocation for frequent "+OK" response.
		return okReply, nil
	case len(line) == 4 && line[0] == 'P' && line[1] == 'O' && line[2] == 'N' && line[3] == 'G':
		// Avoid allocation in PING command benchmarks :)
		return pongReply, nil
	default:
		return RESP_SIMPLE_STRING(string(line[0:])), nil
	}
}

func parseError(line []byte) (RESP_ERROR, error) {
	return RESP_ERROR(string(line[0:])), nil
}

// parseInt parses an integer reply.
func parseInt(p []byte) (RESP_INTEGER, error) {
	if len(p) == 0 {
		return 0, errors.New("malformed integer")
	}

	var negate bool
	if p[0] == '-' {
		negate = true
		p = p[1:]
		if len(p) == 0 {
			return 0, errors.New("malformed integer")
		}
	}

	var n int64
	for _, b := range p {
		n *= 10
		if b < '0' || b > '9' {
			return 0, errors.New("illegal bytes in length")
		}
		n += int64(b - '0')
	}

	if negate {
		n = -n
	}
	return RESP_INTEGER(n), nil
}

//当error为ErrUnexpectedRESPEOF时，说明还缺少数据，并不是一个完整的bulk string
//还缺少的数据由第二个参数返回,其余n的值则都是0
func parseBulkString(line []byte, br *bufio.Reader) (RESP_BULK_STRING, int, error) {
	neededDataLen := 0
	n, err := parseLen(line[0:])
	if n < 0 || err != nil {
		return "", neededDataLen, err
	}
	brSize := br.Buffered()
	//n+2： bulkString + "\r\n"
	if n+2 > brSize {
		neededDataLen = n + 2 - brSize
		return "", neededDataLen, ErrUnexpectedRESPEOF
	}
	p, _ := br.Peek(n)
	br.Discard(n)
	if line, err := readLine(br); err != nil {
		return "", neededDataLen, err
	} else if len(line) != 0 {
		return "", neededDataLen, errors.New("bad bulk string format")
	}
	return RESP_BULK_STRING(p), neededDataLen, nil
}

//解析array型数据,当错误类型为ErrUnexpectedRESPEOF，表示Array Data内的bulk string数
//据并不全，还缺少多少长度的数据由第二个参数返回
func parseArrayData(br *bufio.Reader, arrayNum int) (RESP_ARRAY, int, error) {
	r := make([]interface{}, 0)
	neededDataLen := 0
	for i := 0; i < arrayNum; i++ {
		if data, neededDataLen, err := parseRESP(br); err != nil {
			if ErrUnexpectedRESPEOF == err {
				return nil, neededDataLen, err
			} else {
				errString := fmt.Sprintf("bad arrayData format: %s", err.Error())
				return nil, neededDataLen, errors.New(errString)
			}
		} else {
			r = append(r, data)
		}
	}
	return r, neededDataLen, nil
}

//解析RESP协议
func parseRESP(br *bufio.Reader) (interface{}, int, error) {
	neededDataLen := 0
	line, err := readLine(br)
	if err != nil {
		return nil, neededDataLen, err
	}
	if len(line) == 0 {
		return nil, neededDataLen, errors.New("short resp line")
	}

	switch line[0] {
	case '+':
		respSimpleString, err2 := parseSimpleString(line[1:])
		return respSimpleString, neededDataLen, err2
	case '-':
		respError, err2 := parseError(line[1:])
		return respError, neededDataLen, err2
	case ':':
		respInt, err2 := parseInt(line[1:])
		return respInt, neededDataLen, err2
	case '$':
		return parseBulkString(line[1:], br)
	case '*':
		n, err := parseLen(line[1:])
		if n < 0 || err != nil {
			return nil, neededDataLen, err
		}
		return parseArrayData(br, n)
	}
	return nil, neededDataLen, errors.New("unexpected redis data")
}

func printRESPArray(data RESP_ARRAY) {
	fmt.Println("SUB RESP_ARRAY DATA BEGIN")
	for _, tmp := range data {
		printRESP(tmp)
	}
	fmt.Println("SUB RESP_ARRAY DATA END")
}

func printRESP(data interface{}) {
	switch data.(type) {
	case RESP_ARRAY:
		fmt.Printf("RESP_ARRAY %v\n", data)
		printRESPArray(data.(RESP_ARRAY))
		break
	case RESP_SIMPLE_STRING:
		fmt.Printf("RESP_SIMPLE_STRING %s\n", data.(RESP_SIMPLE_STRING))
		break
	case RESP_INTEGER:
		fmt.Printf("RESP_INTEGER %d\n", data)
		break
	case RESP_ERROR:
		fmt.Printf("RESP_ERROR %s\n", data)
	case RESP_BULK_STRING:
		fmt.Printf("RESP_BULK_STRING %s\n", data.(RESP_BULK_STRING))
	default:
		fmt.Println("unknown type")
	}
}
