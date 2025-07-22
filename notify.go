package smpp

var BoundNotify func(BindInfo) = nil
var TraceMessage func(Direction, CommandID, StatusCode, uint32, []byte)

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
