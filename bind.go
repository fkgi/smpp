package smpp

import (
	"errors"
	"net"
	"time"

	"github.com/fkgi/teldata"
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
	TypeOfNumber  teldata.NatureOfAddress
	NumberingPlan teldata.NumberingPlan
	AddressRange  string
}

type Bind struct {
	BindInfo
	con      net.Conn
	ver      byte
	eventQ   chan message
	reqStack map[uint32]chan message
	sequence chan uint32
}

func (b *Bind) nextSequence() uint32 {
	ret := <-b.sequence
	if ret == 0x7fffffff {
		b.sequence <- 1
	} else {
		b.sequence <- ret + 1
	}
	return ret
}

func (b *Bind) ListenAndServe(c net.Conn) (e error) {
	b.con = c
	b.sequence = make(chan uint32, 1)
	b.sequence <- 1
	b.ver = 0x34

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
		c.Close()
		e = errors.New("invalid request for binding")
		return
	}

	req := bindReq{cmd: msg.id}
	res := bindRes{
		cmd:      msg.id | GenericNack,
		SystemID: ID,
		Version:  b.ver}

	if e = req.Unmarshal(msg.body); e != nil {
		b.writePDU(message{
			id:   res.CommandID(),
			stat: StatBindFail,
			seq:  msg.seq})
		c.Close()
		return
	}

	b.PeerID = req.SystemID
	b.Password = req.Password
	b.SystemType = req.SystemType
	b.TypeOfNumber = req.AddrTON
	b.NumberingPlan = req.AddrNPI
	b.AddressRange = req.AddrRange

	if req.Version < b.ver {
		b.ver = req.Version
	}

	if e = b.writePDU(message{
		id:   res.CommandID(),
		stat: StatOK,
		seq:  msg.seq,
		body: res.Marshal(b.ver)}); e != nil {
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
	b.sequence = make(chan uint32, 1)
	b.sequence <- 1
	b.ver = 0x34

	defer func() {
		if e != nil {
			c.Close()
		}
	}()

	req := bindReq{
		SystemID:   ID,
		Password:   b.Password,
		SystemType: b.SystemType,
		Version:    b.ver,
		AddrTON:    b.TypeOfNumber,
		AddrNPI:    b.NumberingPlan,
		AddrRange:  b.AddressRange}
	switch b.BindType {
	case RxBind:
		req.cmd = BindReceiver
	case TxBind:
		req.cmd = BindTransmitter
	case TRxBind:
		req.cmd = BindTransceiver
	default:
		return errors.New("invalid bind type")
	}
	seq := b.nextSequence()

	if e = b.writePDU(message{
		id:   req.CommandID(),
		seq:  seq,
		body: req.Marshal(b.ver)}); e != nil {
		return
	}

	msg, e := b.readPDU()
	if e != nil {
		return
	}

	if msg.id != req.cmd|GenericNack {
		e = errors.New("invalid response")
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

	res := bindRes{cmd: msg.id}
	if e = res.Unmarshal(msg.body); e != nil {
		return
	}
	b.PeerID = res.SystemID
	if res.Version < b.ver {
		b.ver = res.Version
	}

	if BoundNotify != nil {
		BoundNotify(b.BindInfo)
	}

	return b.serve()
}

func (b *Bind) Close() {
	if b.reqStack == nil {
		return
	}

	msg := message{
		id:       Unbind,
		seq:      b.nextSequence(),
		callback: make(chan message)}
	b.eventQ <- msg

	wt := time.AfterFunc(Expire, func() {
		b.eventQ <- message{
			id:   internalFailure,
			stat: 0xFFFFFFFF,
			seq:  msg.seq}
	})
	msg = <-msg.callback
	wt.Stop()

	b.con.Close()
}

func (b *Bind) Send(r PDU) (s StatusCode, a PDU, e error) {
	if b.reqStack == nil {
		e = errors.New("closed bind")
		return
	}

	msg := message{
		id:       r.CommandID(),
		seq:      b.nextSequence(),
		body:     r.Marshal(b.ver),
		callback: make(chan message)}
	b.eventQ <- msg

	wt := time.AfterFunc(Expire, func() {
		b.eventQ <- message{
			id:   internalFailure,
			stat: 0xFFFFFFFF,
			seq:  msg.seq}
	})
	msg = <-msg.callback
	wt.Stop()

	s = msg.stat
	switch msg.id {
	case SubmitSmResp, DeliverSmResp, DataSmResp, GenericNack:
		a = MakePDUof(msg.id)
		e = a.Unmarshal(msg.body)
	case internalFailure:
		e = errors.New("request timeout")
	default:
		e = errors.New("unexpected response")
	}
	return
}

func (b *Bind) IsActive() bool {
	return b.reqStack != nil
}
