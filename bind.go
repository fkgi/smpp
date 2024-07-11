package smpp

import (
	"bytes"
	"errors"
	"io"
	"net"
	"time"
)

type bindtype int

const (
	ClosedBind bindtype = iota
	TxBind
	RxBind
	TRxBind
)

type BindInfo struct {
	BindType bindtype
	PeerID   string

	Password      string
	SystemType    string
	TypeOfNumber  byte
	NumberingPlan byte
	AddressRange  string
}

type Bind struct {
	BindInfo
	con      net.Conn
	eventQ   chan message
	msgQ     chan message
	reqStack map[uint32]chan message
}

func Accept(l net.Listener) (b Bind, e error) {
	if b.con, e = l.Accept(); e != nil {
		return
	}

	msg, e := b.readPDU()
	if e != nil {
		return
	}
	switch msg.id {
	case 0x00000001: // bind_receiver
		b.BindType = TxBind
	case 0x00000002: // bind_transmitter
		b.BindType = RxBind
	case 0x00000009: // bind_transceiver
		b.BindType = TRxBind
	default:
		b.writePDU(0x80000000, 0x00000003, msg.seq, nil)
		e = errors.New("invalid request")
		return
	}
	id := msg.id | 0x80000000

	// verify request
	buf := bytes.NewBuffer(msg.body)
	var tmp byte
	if b.PeerID, e = readCString(buf); e != nil {
	} else if b.Password, e = readCString(buf); e != nil {
	} else if b.SystemType, e = readCString(buf); e != nil {
	} else if tmp, e = buf.ReadByte(); e != nil {
	} else if tmp != 0x34 {
		e = errors.New("invalid version")
	} else if b.TypeOfNumber, e = buf.ReadByte(); e != nil {
	} else if b.NumberingPlan, e = buf.ReadByte(); e != nil {
	} else {
		b.AddressRange, e = readCString(buf)
	}

	if e != nil {
		b.writePDU(0x80000000, 0x0000000d, msg.seq, nil)
		return
	}

	// make response
	w := bytes.Buffer{}
	// system_id
	writeCString([]byte(ID), &w)
	// interface_version
	writeTLV(0x0210, []byte{0x34}, &w)

	if e = b.writePDU(id, 0, msg.seq, w.Bytes()); e != nil {
		return
	}

	b.eventQ = make(chan message, 1024)
	b.msgQ = make(chan message, 1024)
	b.reqStack = make(map[uint32]chan message)

	go b.serve()
	return
}

func Connect(c net.Conn, info BindInfo) (b Bind, e error) {
	b.BindInfo = info
	b.con = c

	var id uint32
	switch b.BindType {
	case RxBind:
		id = 0x00000001
	case TxBind:
		id = 0x00000002
	case TRxBind:
		id = 0x00000009
	default:
		e = errors.New("invalid bind type")
		return
	}

	seq := nextSequence()

	w := bytes.Buffer{}
	// system_id
	writeCString([]byte(ID), &w)
	// password
	writeCString([]byte(b.Password), &w)
	// system_type
	writeCString([]byte(b.SystemType), &w)
	// interface_version
	w.WriteByte(0x34)
	// addr_ton
	w.WriteByte(b.TypeOfNumber)
	// addr_npi
	w.WriteByte(b.NumberingPlan)
	// address_range
	writeCString([]byte(b.AddressRange), &w)

	if e = b.writePDU(id, 0, seq, w.Bytes()); e != nil {
		return
	}

	msg, e := b.readPDU()
	if e != nil {
		return
	}

	switch msg.id {
	case 0x80000001: // bind_receiver_resp
		if info.BindType != RxBind {
			e = errors.New("invalid response")
		}
	case 0x80000002: // bind_transmitter_resp
		if info.BindType != TxBind {
			e = errors.New("invalid response")
		}
	case 0x80000009: // bind_transceiver_resp
		if info.BindType != TRxBind {
			e = errors.New("invalid response")
		}
	default:
		e = errors.New("invalid response")
	}
	if e != nil {
		return
	}

	if msg.stat != 0 {
		e = errors.New("error response")
		return
	}

	if seq != msg.seq {
		e = errors.New("invalid sequence")
		return
	}

	// verify request
	buf := bytes.NewBuffer(msg.body)
	var peerVersion byte
	b.PeerID, e = readCString(buf)
	if e != nil {
		return
	}
	for {
		t, v, e2 := readTLV(buf)
		if e2 == io.EOF {
			break
		}
		if e2 != nil {
			e = e2
			break
		}

		switch t {
		case 0x0210:
			peerVersion = v[0]
		}
	}
	if peerVersion != 0 && peerVersion != 0x34 {
		e = errors.New("invalid version")
		c.Close()
		return
	}

	b.eventQ = make(chan message, 1024)
	b.msgQ = make(chan message, 1024)
	b.reqStack = make(map[uint32]chan message)

	go b.serve()
	return
}

func (b *Bind) Close() {
	msg := message{
		id:     0x00000006,
		seq:    nextSequence(),
		notify: make(chan message)}
	b.eventQ <- msg

	wt := time.AfterFunc(Expire, func() {
		b.eventQ <- message{
			id:   0x80000000,
			stat: 0xFFFFFFFF,
			seq:  msg.seq}
	})
	msg = <-msg.notify
	wt.Stop()

	b.con.Close()
}

func (b *Bind) Send(p Request) (e error) {
	msg := message{
		id:     p.CommandID(),
		seq:    nextSequence(),
		body:   p.Marshal(),
		notify: make(chan message)}
	b.eventQ <- msg

	wt := time.AfterFunc(Expire, func() {
		b.eventQ <- message{
			id:   0x80000000,
			stat: 0xFFFFFFFF,
			seq:  msg.seq}
	})
	msg = <-msg.notify
	wt.Stop()

	if msg.stat != 0x00000000 {
		e = errors.New("send failed")
	}
	return
}
