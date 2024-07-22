package smpp

import (
	"fmt"
	"math/rand"
	"time"
)

type message struct {
	id   CommandID
	stat StatusCode
	seq  uint32
	body []byte

	callback chan message
}

func (b *Bind) serve() {

	enquireT := time.AfterFunc(KeepAlive, func() {
		msg := message{
			id:       EnquireLink,
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
		if msg.stat != 0x00000000 {
			b.Close()
		}
	})

	// worker for event
	go func() {
		for {
			msg := <-b.eventQ
			var e error

			if msg.callback != nil {
				// Tx event
				if msg.id < 0x80000000 {
					// Tx req
					b.reqStack[msg.seq] = msg.callback
					e = b.writePDU(msg)
				} else {
					// Tx ans
					e = b.writePDU(msg)
				}
			} else {
				// Rx event
				if msg.id == CloseConnection {
					break
				} else if msg.id == EnquireLink {
					e = b.writePDU(message{
						id:  EnquireLinkResp,
						seq: msg.seq})
				} else if msg.id == Unbind {
					b.writePDU(message{
						id:  UnbindResp,
						seq: msg.seq})
					b.con.Close()
				} else if msg.id < 0x80000000 {
					// Rx other req
					e = b.writePDU(message{
						id:   GenericNack,
						stat: StatInvCmdID,
						seq:  msg.seq})
				} else if callback, ok := b.reqStack[msg.seq]; ok {
					// Handle Rx ans
					delete(b.reqStack, msg.seq)
					callback <- msg
				}
			}

			if e == nil {
				enquireT.Reset(KeepAlive)
			} else {
				b.con.Close()
			}
		}
	}()

	// worker for Rx request handling
	go func() {
		for msg, ok := <-b.requestQ; ok; msg, ok = <-b.requestQ {
			var req Request
			switch msg.id {
			case SubmitSm:
				req = &SubmitSM{}
			case DeliverSm:
				req = &DeliverSM{}
			case DataSm:
				req = &DataSM{}
			}
			if req == nil {
				panic(fmt.Sprintf("unexpected request PDU (ID:%#x)", msg.id))
			}

			var res Response
			if e := req.Unmarshal(msg.body); e != nil {
				res = req.MakeResponse(StatSysErr)
			} else if res = RequestHandler(b.BindInfo, req); res == nil {
				res = req.MakeResponse(StatSysErr)
			}
			b.eventQ <- message{
				id:       res.CommandID(),
				stat:     res.CommandStatus(),
				seq:      msg.seq,
				body:     res.Marshal(),
				callback: make(chan message)}
		}
	}()

	// worker for Rx data from socket
	for msg, e := b.readPDU(); e == nil; msg, e = b.readPDU() {
		switch msg.id {
		// case QuerySm:
		case SubmitSm, DeliverSm, DataSm:
			b.requestQ <- msg
		// case ReplaceSm:
		// case CancelSm:
		// case Outbind:
		// case SubmitMulti:
		default:
			b.eventQ <- msg
		}
	}

	enquireT.Stop()
	b.BindType = NilBind
	b.con.Close()
	close(b.requestQ)
	b.eventQ <- message{id: CloseConnection}
	// close(b.eventQ)

	if ConnectionDownNotify != nil {
		ConnectionDownNotify(b)
	}
}

/*
var HandleSubmit func(info BindInfo, pdu SubmitSM) (uint32, SubmitSM_resp) = nil
var HandleDeliver func(info BindInfo, pdu DeliverSM) (uint32, DeliverSM_resp) = nil
var HandleData func(info BindInfo, pdu DataSM) (uint32, DataSM_resp) = nil
*/

var RequestHandler = func(info BindInfo, pdu Request) Response {
	const l = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	id := make([]byte, 16)
	for i := range id {
		id[i] = l[rand.Intn(len(l))]
	}
	switch pdu.(type) {
	case *DataSM:
		return &DataSM_resp{
			MessageID: string(id),
		}
	case *SubmitSM:
		return &SubmitSM_resp{
			MessageID: string(id),
		}
	case *DeliverSM:
		return &DeliverSM_resp{}
	}
	return nil
}
