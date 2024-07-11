package smpp

import (
	"time"
)

var (
	ID        = ""
	WhiteList []BindInfo
	KeepAlive = time.Second * 2
	Expire    = time.Second

	sequence = make(chan uint32, 1)
)

func init() {
	sequence <- 1
}

func nextSequence() uint32 {
	ret := <-sequence
	if ret == 0x7fffffff {
		sequence <- 1
	} else {
		sequence <- ret + 1
	}
	return ret
}
