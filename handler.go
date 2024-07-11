package smpp

import "time"

type message struct {
	id   uint32
	stat uint32
	seq  uint32
	body []byte

	notify chan message
}

func (b *Bind) serve() {
	// worker for Rx data from socket
	go func() {
		for msg, e := b.readPDU(); e == nil; msg, e = b.readPDU() {
			switch msg.id {
			case 0x00000004: // submit_sm
				b.msgQ <- msg
			case 0x00000005: // deliver_sm
				b.msgQ <- msg
			case 0x00000103: // data_sm
				b.msgQ <- msg
			default:
				b.eventQ <- msg
			}
		}
	}()

	// worker for Rx request handling
	go func() {
		for msg, ok := <-b.msgQ; ok; msg, ok = <-b.msgQ {
			var d Request
			switch msg.id {
			case 0x00000004: // submit_sm
				d = &SubmitSM{}
			case 0x00000005: // deliver_sm
				d = &DeliverSM{}
			case 0x00000103: // data_sm
				d = &DataSM{}
			}
			d.Unmarshal(msg.body)

			var res Response
			msg.stat, res = RequestHandler(b.BindInfo, d)
			if res != nil {
				msg.id = res.CommandID()
				msg.body = res.Marshal()
			} else {
				msg.id = 0x80000000
				msg.body = nil
			}
			msg.notify = make(chan message)
			b.eventQ <- msg
		}
	}()

	enquireT := time.AfterFunc(KeepAlive, func() {
		msg := message{
			id:     0x00000015,
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
		if msg.stat != 0x00000000 {
			b.Close()
		}
	})

	for m, ok := <-b.eventQ; ok; m, ok = <-b.eventQ {
		if m.notify != nil && m.id&0x80000000 == 0x00000000 { // Tx req
			b.reqStack[m.seq] = m.notify
			e := b.writePDU(m.id, 0, m.seq, m.body)
			if e != nil {
				break
			}
		} else if m.notify != nil { // Tx ans
			b.writePDU(m.id, m.stat, m.seq, m.body)
		} else if m.id == 0x00000015 { // Rx enquire_link
			e := b.writePDU(0x80000015, 0, m.seq, nil)
			if e != nil {
				break
			}
		} else if m.id == 0x00000006 { // Rx unbind
			b.writePDU(0x80000006, 0, m.seq, nil)
			break
		} else if m.id < 0x80000000 { // Rx other req
			e := b.writePDU(0x80000000, 0x00000003, m.seq, nil)
			if e != nil {
				break
			}
		} else { // Handle Rx ans
			notify, ok := b.reqStack[m.seq]
			if ok {
				notify <- m
			}
		}
		enquireT.Reset(KeepAlive)
	}

	enquireT.Stop()
	b.con.Close()
}

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
		return 0, &DeliverSM_resp{
			MessageID: "random id",
		}
	}
	return 0x00000003, nil
}
