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
	NilBind bindtype = iota
	TxBind
	RxBind
	TRxBind
)

func (t bindtype) String() string {
	switch t {
	case TxBind:
		return "Tx"
	case RxBind:
		return "Rx"
	case TRxBind:
		return "TRx"
	default:
		return "undefined"
	}
}

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
	reqStack map[uint32]chan message
}

func (b *Bind) ListenAndServe(c net.Conn) (e error) {
	b.con = c

	msg, e := b.readPDU()
	if e != nil {
		c.Close()
		return
	}
	switch msg.id {
	case BindReceiver:
		b.BindType = TxBind
	case BindTransmitter:
		b.BindType = RxBind
	case BindTransceiver:
		b.BindType = TRxBind
	default:
		b.writePDU(message{
			id:   GenericNack,
			stat: StatInvCmdID,
			seq:  msg.seq})
		e = errors.New("invalid request for binding")
		c.Close()
		return
	}
	id := msg.id | GenericNack

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
		b.writePDU(message{
			id:   GenericNack,
			stat: StatBindFail,
			seq:  msg.seq})
		c.Close()
		return
	}

	w := new(bytes.Buffer)
	// system_id
	writeCString([]byte(ID), w)
	// interface_version
	writeTLV(0x0210, []byte{0x34}, w)

	if e = b.writePDU(message{
		id:   id,
		seq:  msg.seq,
		body: w.Bytes()}); e != nil {
		c.Close()
		return
	}

	if BoundNotify != nil {
		BoundNotify(b.BindInfo)
	}

	return b.serve()
}

func (b *Bind) DialAndServe(c net.Conn) (e error) {
	b.con = c

	var id CommandID
	switch b.BindType {
	case RxBind:
		id = BindReceiver
	case TxBind:
		id = BindTransmitter
	case TRxBind:
		id = BindTransceiver
	default:
		return errors.New("invalid bind type")
	}

	seq := nextSequence()

	w := new(bytes.Buffer)
	// system_id
	writeCString([]byte(ID), w)
	// password
	writeCString([]byte(b.Password), w)
	// system_type
	writeCString([]byte(b.SystemType), w)
	// interface_version
	w.WriteByte(0x34)
	// addr_ton
	w.WriteByte(b.TypeOfNumber)
	// addr_npi
	w.WriteByte(b.NumberingPlan)
	// address_range
	writeCString([]byte(b.AddressRange), w)

	if e = b.writePDU(message{
		id:   id,
		seq:  seq,
		body: w.Bytes()}); e != nil {
		return
	}

	msg, e := b.readPDU()
	if e != nil {
		return
	}

	switch msg.id {
	case BindReceiverResp:
		if b.BindType != RxBind {
			e = errors.New("invalid response")
		}
	case BindTransmitterResp:
		if b.BindType != TxBind {
			e = errors.New("invalid response")
		}
	case BindTransceiverResp:
		if b.BindType != TRxBind {
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

	// verify response
	buf := bytes.NewBuffer(msg.body)
	var peerVersion byte
	if b.PeerID, e = readCString(buf); e != nil {
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

	if BoundNotify != nil {
		BoundNotify(b.BindInfo)
	}

	return b.serve()
}

func (b *Bind) Close() {
	if b.BindType == NilBind {
		return
	}

	msg := message{
		id:       Unbind,
		seq:      nextSequence(),
		callback: make(chan message)}
	b.eventQ <- msg

	wt := time.AfterFunc(Expire, func() {
		b.eventQ <- message{
			id:   InternalFailure,
			stat: 0xFFFFFFFF,
			seq:  msg.seq}
	})
	msg = <-msg.callback
	wt.Stop()

	b.con.Close()
}

func (b *Bind) Send(p Request) (a Response, e error) {
	if b.BindType == NilBind {
		e = errors.New("closed bind")
		return
	}

	msg := message{
		id:       p.CommandID(),
		seq:      nextSequence(),
		body:     p.Marshal(),
		callback: make(chan message)}
	b.eventQ <- msg

	wt := time.AfterFunc(Expire, func() {
		b.eventQ <- message{
			id:   InternalFailure,
			stat: 0xFFFFFFFF,
			seq:  msg.seq}
	})
	msg = <-msg.callback
	wt.Stop()

	switch msg.id {
	case SubmitSmResp:
		a = &SubmitSM_resp{Status: msg.stat}
	case DeliverSmResp:
		a = &DeliverSM_resp{Status: msg.stat}
	case DataSmResp:
		a = &DataSM_resp{Status: msg.stat}
	case GenericNack:
		e = errors.New("send failed")
	case InternalFailure:
		e = errors.New("request timeout")
	default:
		e = errors.New("unexpected response")
	}
	if e == nil {
		e = a.Unmarshal(msg.body)
	}

	return
}
