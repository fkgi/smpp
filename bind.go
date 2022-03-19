package smpp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"net"
	"time"
)

var (
	ID        = ""
	WhiteList []BindInfo
	KeepAlive = time.Minute
)

type state int

const (
	open state = iota
	boundTx
	bountRx
	bountTRx
	closed
)

type event struct {
	trx       bool
	commandID uint32
	result    uint32
	sequence  uint32
	body      []byte
}

type sequence chan uint32

func newSequence() sequence {
	seq := make(chan uint32, 1)
	seq <- 1
	return seq
}

func (s sequence) next() uint32 {
	ret := <-s
	if ret == 0x7fffffff {
		s <- 1
	} else {
		s <- ret + 1
	}
	return ret
}

type bindtype int

const (
	AnyBind bindtype = iota
	TxBind
	RxBind
	TRxBind
)

type BindInfo struct {
	BindType      bindtype
	SystemID      string
	Password      string
	SystemType    string
	TypeOfNumber  byte
	NumberingPlan byte
	AddressRange  string
}

type Bind struct {
	BindInfo
	con net.Conn
	sequence
	state
	eventQ chan event
}

func (b *Bind) readPDU() (id, stat, num uint32, body []byte, e error) {
	var l uint32
	if e = binary.Read(b.con, binary.BigEndian, &l); e != nil {
		return
	}
	if l < 16 {
		e = errors.New("invalid header")
		return
	}
	l -= 16
	if e = binary.Read(b.con, binary.BigEndian, &id); e != nil {
		return
	}
	if e = binary.Read(b.con, binary.BigEndian, &stat); e != nil {
		return
	}
	if e = binary.Read(b.con, binary.BigEndian, &num); e != nil {
		return
	}
	if l != 0 {
		body = make([]byte, l)
		offset := 0
		n := 0
		for offset < 1 {
			n, e = b.con.Read(body[offset:])
			offset += n
			if e != nil {
				break
			}
		}
	}
	return
}

func (b *Bind) writePDU(id, stat, num uint32, body []byte) (e error) {
	if body == nil {
		body = []byte{}
	}
	buf := bufio.NewWriter(b.con)

	// command_length
	binary.Write(buf, binary.BigEndian, uint32(len(body)+16))
	// command_id
	binary.Write(buf, binary.BigEndian, id)
	// command_status
	binary.Write(buf, binary.BigEndian, stat)
	// sequence_number
	binary.Write(buf, binary.BigEndian, num)

	buf.Write(body)
	return buf.Flush()
}
