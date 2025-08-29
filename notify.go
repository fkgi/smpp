package smpp

import "net"

var BoundNotify func(BindInfo, net.Addr) = nil
var UnboundNotify func(BindInfo, net.Addr) = nil
var TraceMessage func(Direction, CommandID, StatusCode, uint32, []byte) = nil

type Direction bool

func (v Direction) String() string {
	if v {
		return "Tx"
	}
	return "Rx"
}

const (
	Tx Direction = true
	Rx Direction = false
)
