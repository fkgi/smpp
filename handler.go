package smpp

import (
	"time"
)

type message struct {
	id   CommandID
	stat uint32
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
					e = b.writePDU(msg.id, 0, msg.seq, msg.body)
				} else {
					// Tx ans
					e = b.writePDU(msg.id, msg.stat, msg.seq, msg.body)
				}
			} else {
				// Rx event
				if msg.id == CloseConnection {
					break
				} else if msg.id == EnquireLink {
					e = b.writePDU(EnquireLinkResp, 0, msg.seq, nil)
				} else if msg.id == Unbind {
					b.writePDU(UnbindResp, 0, msg.seq, nil)
					b.con.Close()
				} else if msg.id < 0x80000000 {
					// Rx other req
					e = b.writePDU(GenericNack, 0x00000003, msg.seq, nil)
				} else if callback, ok := b.reqStack[msg.seq]; ok {
					// Handle Rx ans
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
			var d Request
			switch msg.id {
			case SubmitSm:
				d = &SubmitSM{}
			case DeliverSm:
				d = &DeliverSM{}
			case DataSm:
				d = &DataSM{}
			}

			if e := d.Unmarshal(msg.body); e != nil {
				msg.id = GenericNack
				msg.stat = 0x00000008
				msg.body = nil
			} else if stat, res := RequestHandler(b.BindInfo, d); res == nil {
				msg.id = GenericNack
				msg.stat = 0x00000008
				msg.body = nil
			} else {
				msg.id = res.CommandID()
				msg.body = res.Marshal()
				msg.stat = stat
			}
			msg.callback = make(chan message)
			b.eventQ <- msg
		}
	}()

	// worker for Rx data from socket
	for msg, e := b.readPDU(); e == nil; msg, e = b.readPDU() {
		switch msg.id {
		// case QuerySm:
		case SubmitSm:
			b.requestQ <- msg
		case DeliverSm:
			b.requestQ <- msg
		// case ReplaceSm:
		// case CancelSm:
		// case Outbind:
		// case SubmitMulti:
		case DataSm:
			b.requestQ <- msg
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

var RequestHandler = func(info BindInfo, pdu Request) (uint32, Response) {
	switch pdu.(type) {
	case *DataSM:
		return 0, &DataSM_resp{
			MessageID: "random id",
		}
	case *SubmitSM:
		return 0, &SubmitSM_resp{
			MessageID: "random id",
		}
	case *DeliverSM:
		return 0, &DeliverSM_resp{}
	}
	return 0, nil
}
