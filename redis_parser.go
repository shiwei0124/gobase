// redis_parser.go
package gobase

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
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

type RedisCmd struct {
	br *bufio.Reader
}

func NewRedisCmd(data []byte) *RedisCmd {
	reader := bytes.NewReader(data)
	br := bufio.NewReader(reader)
	return &RedisCmd{
		br: br,
	}
}

func (c *RedisCmd) ParseRequest() (interface{}, error) {
	if req, err := parseRESP(c.br); err != nil {
		if err == ErrUnexpectedRESPEOF {
			return nil, err
		}
		return nil, errors.New(fmt.Sprintf("parse redis request failed, err: %s", err.Error()))
	} else {
		return req, nil
	}
}

func (c *RedisCmd) ParseResponse() (interface{}, error) {
	if resp, err := parseRESP(c.br); err != nil {
		if err == ErrUnexpectedRESPEOF {
			return nil, err
		}
		return nil, errors.New(fmt.Sprintf("parse redis response failed, err: %s", err.Error()))
	} else {
		return resp, nil
	}
}

func readLine(br *bufio.Reader) ([]byte, error) {
	p, err := br.ReadSlice('\n')
	if err == bufio.ErrBufferFull {
		return nil, ErrUnexpectedRESPEOF
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

func parseBulkString(line []byte, br *bufio.Reader) (RESP_BULK_STRING, error) {
	n, err := parseLen(line[0:])
	if n < 0 || err != nil {
		return "", err
	}
	p := make([]byte, n)
	_, err = io.ReadFull(br, p)
	if err != nil {
		return "", err
	}
	if line, err := readLine(br); err != nil {
		return "", err
	} else if len(line) != 0 {
		return "", errors.New("bad bulk string format")
	}
	return RESP_BULK_STRING(p), nil
}

//解析array型数据
func parseArrayData(br *bufio.Reader, arrayNum int) (RESP_ARRAY, error) {
	r := make([]interface{}, 0)
	for i := 0; i < arrayNum; i++ {
		if data, err := parseRESP(br); err != nil {
			return nil, errors.New("bad arrayData format")
		} else {
			r = append(r, data)
		}
	}
	return r, nil
}

//解析RESP协议
func parseRESP(br *bufio.Reader) (interface{}, error) {
	line, err := readLine(br)
	if err != nil {
		return nil, err
	}
	if len(line) == 0 {
		return nil, errors.New("short resp line")
	}

	switch line[0] {
	case '+':
		return parseSimpleString(line[1:])
	case '-':
		return parseError(line[1:])
	case ':':
		return parseInt(line[1:])
	case '$':
		return parseBulkString(line[1:], br)
	case '*':
		n, err := parseLen(line[1:])
		if n < 0 || err != nil {
			return nil, err
		}
		if arrayData, err := parseArrayData(br, n); err != nil {
			return nil, err
		} else {
			return arrayData, nil
		}
	}
	return nil, errors.New("unexpected redis data")
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
