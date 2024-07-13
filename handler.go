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
		close(b.msgQ)
		// close(b.eventQ)
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

	for {
		msg := <-b.eventQ
		var e error

		if msg.notify != nil && msg.id < 0x80000000 {
			// Tx req
			b.reqStack[msg.seq] = msg.notify
			e = b.writePDU(msg.id, 0, msg.seq, msg.body)
		} else if msg.notify != nil {
			// Tx ans
			e = b.writePDU(msg.id, msg.stat, msg.seq, msg.body)
		} else if msg.id == 0x00000015 {
			// Rx enquire_link
			e = b.writePDU(0x80000015, 0, msg.seq, nil)
		} else if msg.id == 0x00000006 {
			// Rx unbind
			b.writePDU(0x80000006, 0, msg.seq, nil)
			break
		} else if msg.id < 0x80000000 {
			// Rx other req
			e = b.writePDU(0x80000000, 0x00000003, msg.seq, nil)
		} else if notify, ok := b.reqStack[msg.seq]; ok {
			// Handle Rx ans
			notify <- msg
		}

		if e != nil {
			break
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
