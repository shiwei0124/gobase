// pipline.go
package gobase

import (
)

type PipeLine struct {
	Sender		chan interface{} 
	Receiver	chan interface{}
}

func NewPipeLine(SenderCapacity int, ReceiverCapacity int) *PipeLine {
	pipeline := &PipeLine {
		Sender : make(chan interface{}, SenderCapacity),
		Receiver : make(chan interface {}, ReceiverCapacity),
	}
	return pipeline
}

func (p * PipeLine) GetSender() chan <- interface{} {
	return p.Sender
}