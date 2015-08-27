// pipe_line_test.go
package gobase

import (
	"testing"
)

func Test_CreatePipeLine(t * testing.T) {
	p := NewPipeLine(1, 1)
	sender := p.GetSender()
	sender <- 1
	
}
