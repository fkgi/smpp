package smpp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"time"
)

var (
	ID        = ""
	WhiteList []BindInfo
	KeepAlive = time.Second

	sequence = make(chan uint32, 1)
)

func init() {
	sequence <- 1
}

func nextSequence() uint32 {
	ret := <-sequence
	if ret == 0x7fffffff {
		sequence <- 1
	} else {
		sequence <- ret + 1
	}
	return ret
}

type bindtype int

const (
	ClosedBind bindtype = iota
	TxBind
	RxBind
	TRxBind
)

type message struct {
	id     uint32
	stat   uint32
	seq    uint32
	body   []byte
	notify chan message
}

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
	con      net.Conn
	eventQ   chan message
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
	if b.SystemID, e = readCString(buf); e != nil {
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

	go b.serve()
	return
}

func Connect(c net.Conn, info BindInfo) (b Bind, e error) {
	b.BindInfo = info
	b.con = c

	var id uint32
	switch info.BindType {
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
	writeCString([]byte(info.SystemID), &w)
	// password
	writeCString([]byte(info.Password), &w)
	// system_type
	writeCString([]byte(info.SystemType), &w)
	// interface_version
	w.WriteByte(0x34)
	// addr_ton
	w.WriteByte(info.TypeOfNumber)
	// addr_npi
	w.WriteByte(info.NumberingPlan)
	// address_range
	writeCString([]byte(info.AddressRange), &w)

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
	peerSystemID, e := readCString(buf)
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
	log.Println(peerSystemID)
	log.Println(peerVersion)

	go b.serve()

	return
}

func (b *Bind) Close() error {
	return b.writePDU(0x00000006, 0, nextSequence(), nil)
}

func (b *Bind) Send(p PDU) error {
	return b.writePDU(
		p.CommandID(), 0, nextSequence(), p.Marshal())
}

func (b *Bind) serve() {
	b.eventQ = make(chan message, 1024)
	b.reqStack = make(map[uint32]chan message)

	go func() {
		for msg, e := b.readPDU(); e == nil; msg, e = b.readPDU() {
			b.eventQ <- msg
		}
	}()

	enquireT := time.AfterFunc(KeepAlive, func() {
		b.writePDU(0x00000015, 0, nextSequence(), nil)
	})
	for m, ok := <-b.eventQ; ok; m, ok = <-b.eventQ {
		if m.notify != nil {
			// Handle Tx req
			seq := nextSequence()
			b.reqStack[seq] = m.notify
			e := b.writePDU(m.id, 0, seq, m.body)
			if e != nil {
				break
			}
		} else if m.id&0x80000000 == 0x00000000 {
			// Handle Rx req
			var d PDU
			switch m.id {
			case 0x00000005: // deliver_sm
				d = &DeliverSM{}
			case 0x00000103: // data_sm
				d = &DataSM{}
			}

			var e error
			if d == nil {
				e = b.writePDU(0x80000000, 0x00000003, m.seq, nil)
			} else if e = d.Unmarshal(m.body); e != nil {
				e = b.writePDU(d.CommandID()|0x80000000, 0x00000008, m.seq, nil)
			} else {
				m.stat, d = RequestHandler(b.BindInfo, d)
				if d != nil {
					m.body = d.Marshal()
				} else {
					m.body = nil
				}
				e = b.writePDU(d.CommandID(), m.stat, m.seq, m.body)
			}
			if e != nil {
				break
			}
		} else {
			// Handle Rx ans
			notify, ok := b.reqStack[m.seq]
			if ok {
				notify <- m
			}
		}
		enquireT.Reset(KeepAlive)
	}
	enquireT.Stop()
	b.con.Close()

	/*
			switch id {
			case 0x00000015: // enquire_link
				e = b.writePDU(0x80000015, 0, seq, nil)
			case 0x80000015: // enquire_link_resp
			case 0x00000006: // unbind
				e = b.writePDU(0x80000006, 0, seq, nil)
				b.con.Close()
			case 0x80000006: // unbind_resp
				e = errors.New("closed")
				b.con.Close()
			case 0x00000004: // submit_sm
				stat, data = SubmitHandler(b.BindInfo, data)
				e = b.writePDU(0x80000004, stat, seq, data)
			case 0x80000004: // submit_sm_resp
			case 0x00000005: // deliver_sm
				d := DeliverSM{}
				e = d.Unmarshal(data)
				if e != nil {
					stat = 0x00000008
					data = nil
				} else {
					var pdu PDU
					stat, pdu = RequestHandler(b.BindInfo, &d)
					if pdu != nil {
						data = pdu.Marshal()
					} else {
						data = nil
					}
				}
				e = b.writePDU(0x80000005, stat, seq, data)
			case 0x80000005: // deliver_sm_resp
			case 0x00000103: // data_sm
				d := DataSM{}
				e = d.Unmarshal(data)
				if e != nil {
					stat = 0x00000008
					data = nil
				} else {
					var pdu PDU
					stat, pdu = RequestHandler(b.BindInfo, &d)
					if pdu != nil {
						data = pdu.Marshal()
					} else {
						data = nil
					}
				}
				e = b.writePDU(0x80000103, stat, seq, data)
			case 0x80000103: // data_sm_resp
				DataRespHandler(b.BindInfo, stat, data)
			default:
				// generic_nack
				e = b.writePDU(0x80000000, 0x00000003, seq, nil)
			}
			if e != nil {
				break
			}
		}
		return
	*/
}

func (b *Bind) readPDU() (msg message, e error) {
	var l uint32
	if e = binary.Read(b.con, binary.BigEndian, &l); e != nil {
		return
	}
	if l < 16 {
		e = errors.New("invalid header")
		return
	}
	l -= 16
	if e = binary.Read(b.con, binary.BigEndian, &(msg.id)); e != nil {
		return
	}
	if e = binary.Read(b.con, binary.BigEndian, &(msg.stat)); e != nil {
		return
	}
	if e = binary.Read(b.con, binary.BigEndian, &(msg.seq)); e != nil {
		return
	}
	if l != 0 {
		msg.body = make([]byte, l)
		offset := 0
		n := 0
		for offset < 1 {
			n, e = b.con.Read(msg.body[offset:])
			offset += n
			if e != nil {
				break
			}
		}
	}
	return
}

func (b *Bind) writePDU(id, stat, seq uint32, body []byte) (e error) {
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
	binary.Write(buf, binary.BigEndian, seq)

	buf.Write(body)
	return buf.Flush()
}

func readCString(buf *bytes.Buffer) (string, error) {
	b, e := buf.ReadBytes(0x00)
	return string(b), e
}

func writeCString(value []byte, buf *bytes.Buffer) {
	/*
		if id != 0 {
			binary.Write(buf, binary.BigEndian, id)
			binary.Write(buf, binary.BigEndian, uint16(len(value)+1))
		}
	*/
	buf.Write(value)
	buf.WriteByte(0x00)
}

func readTLV(buf *bytes.Buffer) (id uint16, value []byte, e error) {
	var l uint16
	if e = binary.Read(buf, binary.BigEndian, &id); e != nil {
	} else if e = binary.Read(buf, binary.BigEndian, &l); e != nil {
	} else {
		value = make([]byte, int(l))
		_, e = buf.Read(value)
	}
	return
}

func writeTLV(id uint16, value []byte, buf *bytes.Buffer) {
	if id != 0 {
		binary.Write(buf, binary.BigEndian, id)
		binary.Write(buf, binary.BigEndian, uint16(len(value)))
	}
	buf.Write(value)
}
